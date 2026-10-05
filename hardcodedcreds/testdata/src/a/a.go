package a

import "os"

type Config struct {
	Name     string
	Password string
	Token    string
	Endpoint string
	Port     int
}

func lookup() (bool, string) { return false, "" }

// Bad: suspicious variable names holding string literals
var password = "hunter22" // want `potential hardcoded credential in "password"; use environment variable or secret management`

const apiKey = "abcd1234" // want `potential hardcoded credential in "apiKey"`

var (
	dbPasswd     = "letmein" // want `potential hardcoded credential in "dbPasswd"`
	clientSecret = `s3cr3t!` // want `potential hardcoded credential in "clientSecret"`
)

// Bad: only the suspicious name in a multi-name spec is reported
var user, privateKey = "admin", "keydata" // want `potential hardcoded credential in "privateKey"`

// Good: short literals (four bytes including quotes or fewer), empty strings,
// non-string literals, non-literal values and declarations without values
var (
	token      = "abc"
	authHeader = ""
	tokenTTL   = 3600
	secretPath = os.Getenv("SECRET_PATH")
	pwd        string
)

// Bad: string literals that look like secrets, regardless of the name
var (
	awsKeyID  = "AKIAIOSFODNN7EXAMPLE"                                                                         // want `string literal looks like a secret or credential; use environment variable or secret management`
	hexKey    = "0123456789abcdef0123456789abcdef"                                                             // want `string literal looks like a secret or credential`
	jwt       = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U" // want `string literal looks like a secret or credential`
	ghToken   = "ghp_abcdefghijklmnopqrstuvwxyz0123456789"                                                     // want `potential hardcoded credential in "ghToken"` `string literal looks like a secret or credential`
	ghPAT     = "github_pat_ABCDEFGHIJKLMNOPQRSTUV_1234"                                                       // want `string literal looks like a secret or credential`
	header    = "Bearer abcdefghijklmnopqrstuvwxyz"                                                            // want `string literal looks like a secret or credential`
	pemHeader = `-----BEGIN RSA PRIVATE KEY-----`                                                              // want `string literal looks like a secret or credential`
	ecHeader  = "-----BEGIN PRIVATE KEY-----"                                                                  // want `string literal looks like a secret or credential`
)

// Good: ordinary string literals
var greeting = "hello, world"

func assignments(cfg *Config) {
	// Bad: short variable declarations and assignments with suspicious names
	accessKey := "AK-local-dev" // want `potential hardcoded credential in "accessKey"`
	accessKey = "AK-other-dev"  // want `potential hardcoded credential in "accessKey"`

	// Bad: suspicious name and secret-looking value report twice
	credential := "ghp_abcdefghijklmnopqrstuvwxyz0123456789" // want `potential hardcoded credential in "credential"` `string literal looks like a secret or credential`

	// Bad: secret-looking values are reported for any assignment target
	cfg.Endpoint = "Bearer abcdefghijklmnopqrstuvwxyz" // want `string literal looks like a secret or credential`

	// Good: non-suspicious names and non-literal values
	name := "service-name"
	cfg.Name = "service-name"
	ok, secretValue := lookup()
	passwordFromEnv := os.Getenv("PASSWORD")
	apiKeyLen := len(accessKey)

	_, _, _, _, _, _ = name, ok, secretValue, passwordFromEnv, apiKeyLen, credential
}

func literals() []any {
	return []any{
		// Bad: suspicious struct field names holding string literals
		Config{Password: "hunter22"}, // want `potential hardcoded credential in field "Password"; use environment variable or secret management`

		// Bad: secret-looking values in any key-value pair
		Config{Endpoint: "AKIAIOSFODNN7EXAMPLE"},                                  // want `string literal looks like a secret or credential`
		map[string]string{"api": "AKIAIOSFODNN7EXAMPLE"},                          // want `string literal looks like a secret or credential`
		map[string]string{"password": "ghp_abcdefghijklmnopqrstuvwxyz0123456789"}, // want `string literal looks like a secret or credential`

		// Good: short, non-literal and non-string values, and non-suspicious names
		Config{Token: "abc"},
		Config{Token: os.Getenv("TOKEN")},
		Config{Name: "service", Port: 8080},
		map[string]int{"password": 1},
	}
}

// Good: nolint suppresses the diagnostics
var nolintSecret = "hunter22" //nolint:hardcodedcreds

//nolint:golint-sl
var exampleToken = "AKIAIOSFODNN7EXAMPLE"

// Bad: nolint for another analyzer does not suppress
var otherSecret = "hunter22" //nolint:nilcheck // want `potential hardcoded credential in "otherSecret"`

var _ = []any{password, apiKey, dbPasswd, clientSecret, user, privateKey, token, authHeader, tokenTTL, secretPath, pwd,
	awsKeyID, hexKey, jwt, ghToken, ghPAT, header, pemHeader, ecHeader, greeting, nolintSecret, exampleToken, otherSecret}
