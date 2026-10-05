// Package hardcodedcreds detects potential hardcoded credentials and secrets.
//
// Hardcoded credentials are a security risk. This analyzer flags suspicious
// patterns that might be secrets.
package hardcodedcreds

import (
	"encoding/base64"
	"go/ast"
	"go/token"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the hardcodedcreds analyzer's documentation.
const Doc = `detect potential hardcoded credentials and secrets

Hardcoded credentials are a security vulnerability. This analyzer
detects suspicious patterns that might be secrets:

1. String literals assigned to names whose last word names a secret
   (dbPassword, authToken, api_key, clientSecret, PRIVATE_KEY); words
   such as author or tokenizer, and names such as tokenURL or
   passwordPolicy, don't count
2. String literals that look like API keys or tokens (AWS keys, JWTs,
   GitHub tokens, bearer tokens, private key headers, 32 or more hex
   digits other than a checksum written as sha256:... or md5=...)
3. Base64 that decodes to user:password, alone or after "Basic "
4. Connection strings with an embedded password
   (postgres://admin:...@db:5432/app), unless the password is an obvious
   placeholder such as ${DB_PASSWORD}, %s, <password> or xxx

The analyzer checks string literals in variable and constant
declarations, assignments and composite literal key-value pairs, and
skips test files.

Secrets should come from:
- Environment variables
- Secret management systems (Vault, AWS Secrets Manager)
- Kubernetes Secrets`

// Analyzer reports potential hardcoded credentials and secrets.
var Analyzer = &analysis.Analyzer{
	Name:     "hardcodedcreds",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// credentialWords are the name words that denote a credential on their own,
// as the last word of a name: dbPassword, authToken, clientSecret.
var credentialWords = map[string]bool{
	"password": true, "passwd": true, "pwd": true,
	"secret": true, "token": true, "auth": true,
	"credential": true, "credentials": true,
	"apikey": true, "privatekey": true, "accesskey": true, "clientsecret": true,
}

// credentialKeyQualifiers are the words that make a following "key" a
// credential: apiKey, private_key, accessKey, secretKey, signingKey.
var credentialKeyQualifiers = map[string]bool{
	"api": true, "private": true, "access": true, "secret": true, "signing": true,
}

// valueSuffixWords may follow the credential word without changing what the
// name holds: passwordValue, tokenStr, secretB64.
var valueSuffixWords = map[string]bool{
	"value": true, "val": true, "str": true, "string": true,
	"raw": true, "bytes": true, "b64": true, "base64": true, "hex": true,
}

// Patterns that look like secrets
var secretPatterns = []*regexp.Regexp{
	// AWS Access Key ID
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	// JWT tokens
	regexp.MustCompile(`eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`),
	// GitHub tokens
	regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
	regexp.MustCompile(`github_pat_[a-zA-Z0-9_]{22,}`),
	// Generic bearer token
	regexp.MustCompile(`Bearer\s+[a-zA-Z0-9_-]{20,}`),
	// Private key header
	regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`),
}

var (
	// hexKeyPattern is the generic API key pattern: 32 or more hex digits.
	hexKeyPattern = regexp.MustCompile(`[0-9a-fA-F]{32,}`)
	// digestPrefix marks the hex digits after it as a checksum written in
	// digest notation (sha256:..., md5=...), not a key.
	digestPrefix = regexp.MustCompile(`(?i)\b(md5|sha1|sha224|sha256|sha384|sha512)[:=]$`)
	// urlPattern finds URLs in a string literal.
	urlPattern = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s"'<>]+`)
	// base64Pattern matches a padded or unpadded standard base64 token long
	// enough to hold "user:password".
	base64Pattern = regexp.MustCompile(`^[A-Za-z0-9+/]{12,}={0,2}$`)
	// basicAuthPattern matches decoded basic-auth credentials, user:password.
	basicAuthPattern = regexp.MustCompile(`^[A-Za-z0-9._@+-]+:[!-~]{4,}$`)
)

// placeholderPasswords are passwords in connection strings that stand for
// the real one in examples and documentation.
var placeholderPasswords = map[string]bool{
	"password": true, "pass": true, "passwd": true, "pwd": true,
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Build a set of test files to skip
	testFiles := make(map[string]bool)
	for _, f := range pass.Files {
		filename := pass.Fset.Position(f.Pos()).Filename
		if strings.HasSuffix(filename, "_test.go") {
			testFiles[filename] = true
		}
	}

	// Helper to check if a node is in a test file
	isTestFile := func(n ast.Node) bool {
		filename := pass.Fset.Position(n.Pos()).Filename
		return testFiles[filename]
	}

	nodeFilter := []ast.Node{
		(*ast.ValueSpec)(nil),
		(*ast.AssignStmt)(nil),
		(*ast.KeyValueExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		// Skip test files - mock credentials are standard practice in tests
		if isTestFile(n) {
			return
		}

		switch node := n.(type) {
		case *ast.ValueSpec:
			checkValueSpec(reporter, node)
		case *ast.AssignStmt:
			checkAssignment(reporter, node)
		case *ast.KeyValueExpr:
			checkKeyValue(reporter, node)
		}
	})

	return nil, nil
}

func checkValueSpec(reporter *nolint.Reporter, spec *ast.ValueSpec) {
	for i, name := range spec.Names {
		// Check variable name for a string literal value
		if isSuspiciousName(name.Name) && i < len(spec.Values) && isCredentialLiteral(spec.Values[i]) {
			reporter.Reportf(spec.Pos(),
				"potential hardcoded credential in %q; use environment variable or secret management",
				name.Name)
		}

		// Check value for secret patterns
		if i < len(spec.Values) {
			checkExprForSecrets(reporter, spec.Values[i])
		}
	}
}

func checkAssignment(reporter *nolint.Reporter, assign *ast.AssignStmt) {
	for i, lhs := range assign.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}
		if isSuspiciousName(ident.Name) && i < len(assign.Rhs) && isCredentialLiteral(assign.Rhs[i]) {
			reporter.Reportf(assign.Pos(),
				"potential hardcoded credential in %q; use environment variable or secret management",
				ident.Name)
		}
	}

	// Check RHS for secret patterns
	for _, rhs := range assign.Rhs {
		checkExprForSecrets(reporter, rhs)
	}
}

func checkKeyValue(reporter *nolint.Reporter, kv *ast.KeyValueExpr) {
	// Check struct field names
	if ident, ok := kv.Key.(*ast.Ident); ok && isSuspiciousName(ident.Name) && isCredentialLiteral(kv.Value) {
		reporter.Reportf(kv.Pos(),
			"potential hardcoded credential in field %q; use environment variable or secret management",
			ident.Name)
	}

	checkExprForSecrets(reporter, kv.Value)
}

// isCredentialLiteral reports whether expr is a string literal long enough to
// hold a credential.
func isCredentialLiteral(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && len(lit.Value) > 5
}

func checkExprForSecrets(reporter *nolint.Reporter, expr ast.Expr) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return
	}

	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return
	}

	switch {
	case looksLikeSecret(value):
		reporter.Reportf(lit.Pos(),
			"string literal looks like a secret or credential; use environment variable or secret management")
	case hasURLWithPassword(value):
		reporter.Reportf(lit.Pos(),
			"connection string has an embedded password; use environment variable or secret management")
	case hasBase64Credentials(value):
		reporter.Reportf(lit.Pos(),
			"string literal is base64-encoded user:password credentials; use environment variable or secret management")
	}
}

// looksLikeSecret reports whether value matches one of the secret patterns,
// or holds 32 or more hex digits that aren't a checksum in digest notation.
func looksLikeSecret(value string) bool {
	for _, pattern := range secretPatterns {
		if pattern.MatchString(value) {
			return true
		}
	}

	for _, loc := range hexKeyPattern.FindAllStringIndex(value, -1) {
		if !digestPrefix.MatchString(value[:loc[0]]) {
			return true
		}
	}
	return false
}

// hasURLWithPassword reports whether value holds a URL whose user info
// carries a password, such as postgres://admin:hunter2@db:5432/app. Obvious
// placeholders (${DB_PASSWORD}, %s, <password>, xxx, password) don't count.
func hasURLWithPassword(value string) bool {
	for _, raw := range urlPattern.FindAllString(value, -1) {
		u, err := url.Parse(raw)
		if err != nil || u.User == nil {
			continue
		}
		if password, ok := u.User.Password(); ok && !isPlaceholderPassword(password) {
			return true
		}
	}
	return false
}

// isPlaceholderPassword reports whether password stands for a real one.
func isPlaceholderPassword(password string) bool {
	if password == "" || placeholderPasswords[strings.ToLower(password)] {
		return true
	}
	if strings.ContainsAny(password, "${}<>%") {
		return true
	}
	return strings.Trim(password, "xX*.") == ""
}

// hasBase64Credentials reports whether a whitespace-separated word of value,
// such as the one after "Basic ", is base64 that decodes to user:password.
func hasBase64Credentials(value string) bool {
	for word := range strings.FieldsSeq(value) {
		if !base64Pattern.MatchString(word) {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(word)
		if err != nil {
			decoded, err = base64.RawStdEncoding.DecodeString(word)
		}
		if err == nil && basicAuthPattern.Match(decoded) {
			return true
		}
	}
	return false
}

// isSuspiciousName reports whether name, split into words at underscores,
// hyphens and camelCase boundaries, names a credential: its last word is a
// credential word (dbPassword, AUTH_TOKEN) or "key" after a qualifier such
// as api or private (apiKey, PRIVATE_KEY). A trailing word such as Value or
// Str is skipped first (tokenStr). Other words that merely contain a
// credential word (author, tokenizer) or names where the credential word only
// qualifies another (tokenURL, passwordPolicy) don't count.
func isSuspiciousName(name string) bool {
	words := nameWords(name)
	if len(words) > 1 && valueSuffixWords[words[len(words)-1]] {
		words = words[:len(words)-1]
	}
	if len(words) == 0 {
		return false
	}

	last := words[len(words)-1]
	if credentialWords[last] {
		return true
	}
	return last == "key" && len(words) > 1 && credentialKeyQualifiers[words[len(words)-2]]
}

// nameWords splits an identifier into lower-case words at underscores,
// hyphens and camelCase boundaries; an upper-case run followed by a
// lower-case letter ends one letter early, so APIKey is "api", "key".
func nameWords(name string) []string {
	var words []string
	runes := []rune(name)
	start := 0
	flush := func(end int) {
		if end > start {
			words = append(words, strings.ToLower(string(runes[start:end])))
		}
		start = end
	}

	for i, r := range runes {
		switch {
		case r == '_' || r == '-':
			flush(i)
			start = i + 1
		case i > start && unicode.IsUpper(r):
			prev := runes[i-1]
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if !unicode.IsUpper(prev) || nextLower {
				flush(i)
			}
		}
	}
	flush(len(runes))
	return words
}
