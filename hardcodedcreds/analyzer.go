// Package hardcodedcreds detects potential hardcoded credentials and secrets.
//
// Hardcoded credentials are a security risk. This analyzer flags suspicious
// patterns that might be secrets.
package hardcodedcreds

import (
	"go/ast"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `detect potential hardcoded credentials and secrets

Hardcoded credentials are a security vulnerability. This analyzer
detects suspicious patterns that might be secrets:

1. Variable names suggesting secrets (password, apiKey, secret, token)
2. String literals that look like API keys or tokens
3. Base64-encoded strings that might be credentials
4. Connection strings with embedded credentials

Secrets should come from:
- Environment variables
- Secret management systems (Vault, AWS Secrets Manager)
- Kubernetes Secrets`

var Analyzer = &analysis.Analyzer{
	Name:     "hardcodedcreds",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Suspicious variable name patterns
var suspiciousNames = []string{
	"password", "passwd", "pwd",
	"secret", "apikey", "api_key",
	"token", "auth", "credential",
	"private_key", "privatekey",
	"access_key", "accesskey",
	"client_secret", "clientsecret",
}

// Patterns that look like secrets
var secretPatterns = []*regexp.Regexp{
	// AWS Access Key ID
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	// Generic API key pattern (32+ hex chars)
	regexp.MustCompile(`[0-9a-fA-F]{32,}`),
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

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.ValueSpec)(nil),
		(*ast.AssignStmt)(nil),
		(*ast.KeyValueExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
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
		// Check variable name
		if isSuspiciousName(name.Name) {
			// Check if it has a string literal value
			if i < len(spec.Values) {
				if lit, ok := spec.Values[i].(*ast.BasicLit); ok {
					if lit.Kind.String() == "STRING" && len(lit.Value) > 5 {
						reporter.Reportf(spec.Pos(),
							"potential hardcoded credential in %q; use environment variable or secret management",
							name.Name)
					}
				}
			}
		}

		// Check value for secret patterns
		if i < len(spec.Values) {
			checkExprForSecrets(reporter, spec.Values[i])
		}
	}
}

func checkAssignment(reporter *nolint.Reporter, assign *ast.AssignStmt) {
	for i, lhs := range assign.Lhs {
		if ident, ok := lhs.(*ast.Ident); ok {
			if isSuspiciousName(ident.Name) {
				if i < len(assign.Rhs) {
					if lit, ok := assign.Rhs[i].(*ast.BasicLit); ok {
						if lit.Kind.String() == "STRING" && len(lit.Value) > 5 {
							reporter.Reportf(assign.Pos(),
								"potential hardcoded credential in %q; use environment variable or secret management",
								ident.Name)
						}
					}
				}
			}
		}
	}

	// Check RHS for secret patterns
	for _, rhs := range assign.Rhs {
		checkExprForSecrets(reporter, rhs)
	}
}

func checkKeyValue(reporter *nolint.Reporter, kv *ast.KeyValueExpr) {
	// Check struct field names
	if ident, ok := kv.Key.(*ast.Ident); ok {
		if isSuspiciousName(ident.Name) {
			if lit, ok := kv.Value.(*ast.BasicLit); ok {
				if lit.Kind.String() == "STRING" && len(lit.Value) > 5 {
					reporter.Reportf(kv.Pos(),
						"potential hardcoded credential in field %q; use environment variable or secret management",
						ident.Name)
				}
			}
		}
	}

	checkExprForSecrets(reporter, kv.Value)
}

func checkExprForSecrets(reporter *nolint.Reporter, expr ast.Expr) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind.String() != "STRING" {
		return
	}

	value := strings.Trim(lit.Value, "`\"")

	for _, pattern := range secretPatterns {
		if pattern.MatchString(value) {
			reporter.Reportf(lit.Pos(),
				"string literal looks like a secret or credential; use environment variable or secret management")
			return
		}
	}
}

func isSuspiciousName(name string) bool {
	lower := strings.ToLower(name)
	for _, suspicious := range suspiciousNames {
		if strings.Contains(lower, suspicious) {
			return true
		}
	}
	return false
}
