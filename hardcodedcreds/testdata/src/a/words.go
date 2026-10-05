package a

// Good: names that only contain a credential word inside another word, or use
// it to qualify something else, are not credentials
var (
	author         = "Jane Doe"
	tokenizerMode  = "whitespace"
	authorityURL   = "https://auth.example.com"
	tokenURL       = "https://auth.example.com/token"
	passwordPolicy = "strict-mode"
	secretName     = "my-app-secret"
	apiKeyHeader   = "X-API-Key"
	keyboardLayout = "qwertz"
)

// Bad: credential words at the end of camelCase, snake_case and upper-case names
var (
	authToken      = "abc123xyz"    // want `potential hardcoded credential in "authToken"; use environment variable or secret management`
	api_key        = "xyz12345"     // want `potential hardcoded credential in "api_key"`
	API_KEY        = "xyz12345"     // want `potential hardcoded credential in "API_KEY"`
	APIKey         = "xyz12345"     // want `potential hardcoded credential in "APIKey"`
	dbPASSWORD     = "letmein1"     // want `potential hardcoded credential in "dbPASSWORD"`
	signingKey     = "k3y-material" // want `potential hardcoded credential in "signingKey"`
	tokenStr       = "qwerty123"    // want `potential hardcoded credential in "tokenStr"`
	basicAuth      = "admin:admin"  // want `potential hardcoded credential in "basicAuth"`
	oauth2Token    = "tok-123456"   // want `potential hardcoded credential in "oauth2Token"`
	db_credentials = "user:pass1"   // want `potential hardcoded credential in "db_credentials"`
)

// Bad: base64 that decodes to user:password, alone or after "Basic"
var (
	blob        = "dXNlcm5hbWU6c3VwZXJzZWNyZXRwYXNzd29yZA==" // want `string literal is base64-encoded user:password credentials; use environment variable or secret management`
	basicHeader = "Basic YWRtaW46aHVudGVyMjI="               // want `string literal is base64-encoded user:password credentials`
)

// Good: base64 of other text, and words that aren't base64
var (
	logo     = "aGVsbG8gd29ybGQgdGhpcyBpcyBub3QgYSBzZWNyZXQ="
	encoding = "base64-encoded payload"
	pathLike = "abcdefghijklmnop/qrstuvwx"
)

// Bad: connection strings with an embedded password
var (
	dsn      = "postgres://admin:hunter2@db.internal:5432/app" // want `connection string has an embedded password; use environment variable or secret management`
	cacheURL = "redis://:s3cretpw@cache:6379/0"                // want `connection string has an embedded password`
	brokers  = "connect to amqp://guest:guest123@mq:5672/"     // want `connection string has an embedded password`
)

// Good: URLs without a password, or with an obvious placeholder
var (
	envDSN      = "postgres://admin:${DB_PASSWORD}@db.internal:5432/app"
	formatDSN   = "postgres://%s:%s@%s/app"
	exampleDSN  = "postgres://user:password@localhost:5432/app"
	maskedDSN   = "postgres://admin:xxxxxx@db.internal:5432/app"
	angleDSN    = "postgres://admin:<password>@db.internal:5432/app"
	userOnlyURL = "https://user@example.com/"
	plainURL    = "https://example.com/path?q=1"
)

// Good: a checksum in digest notation is not a key
var digest = "sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

// Bad: hex digits without a digest prefix are still reported
var bareHex = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08" // want `string literal looks like a secret or credential`

var _ = []any{author, tokenizerMode, authorityURL, tokenURL, passwordPolicy, secretName, apiKeyHeader, keyboardLayout,
	authToken, api_key, API_KEY, APIKey, dbPASSWORD, signingKey, tokenStr, basicAuth, oauth2Token, db_credentials,
	blob, basicHeader, logo, encoding, pathLike, dsn, cacheURL, brokers,
	envDSN, formatDSN, exampleDSN, maskedDSN, angleDSN, userOnlyURL, plainURL, digest, bareHex}
