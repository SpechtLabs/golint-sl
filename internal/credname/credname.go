// Package credname decides whether an identifier names a credential, from the
// words it is made of. hardcodedcreds uses it for the names that hold
// literals, and dataflow for the parameters it follows into log calls.
package credname

import (
	"strings"
	"unicode"
)

// credentialWords are the name words that denote a credential on their own,
// as the last word of a name: dbPassword, authToken, clientSecret.
var credentialWords = map[string]bool{
	"password": true, "passwd": true, "pwd": true,
	"secret": true, "token": true, "auth": true,
	"credential": true, "credentials": true, "cred": true, "creds": true,
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

// IsCredential reports whether name, split into words at underscores,
// hyphens and camelCase boundaries, names a credential: its last word is a
// credential word (dbPassword, AUTH_TOKEN) or "key" after a qualifier such
// as api or private (apiKey, PRIVATE_KEY). A trailing word such as Value or
// Str is skipped first (tokenStr). Other words that merely contain a
// credential word (author, tokenizer) or names where the credential word only
// qualifies another (tokenURL, passwordPolicy) don't count.
func IsCredential(name string) bool {
	words := Words(name)
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

// Words splits an identifier into lower-case words at underscores,
// hyphens and camelCase boundaries; an upper-case run followed by a
// lower-case letter ends one letter early, so APIKey is "api", "key".
func Words(name string) []string {
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
