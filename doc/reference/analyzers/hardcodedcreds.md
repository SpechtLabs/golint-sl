---
title: hardcodedcreds
permalink: /reference/analyzers/hardcodedcreds
createTime: 2025/01/16 10:00:00
---

Detects potential hardcoded secrets and credentials.

## Category

Architecture

## What It Checks

This analyzer checks the string literals in variable and constant declarations, assignments and composite literal key-value pairs, outside test files, for:

- Values assigned to names whose last word names a secret: `dbPassword`, `authToken`, `api_key`, `clientSecret`, `PRIVATE_KEY`, `tokenStr`. The name is split into words at underscores, hyphens and camelCase boundaries, so `author`, `tokenizerMode`, `tokenURL` and `passwordPolicy` don't count.
- Values that look like API keys or tokens: AWS access key IDs, JWTs, GitHub tokens, bearer tokens, private key headers, and runs of 32 or more hex digits, except a checksum in digest notation such as `sha256:9f86d0...`.
- Base64 that decodes to `user:password`, alone or after `Basic `.
- Connection strings with an embedded password, such as `postgres://admin:hunter2@db:5432/app`. Obvious placeholders (`${DB_PASSWORD}`, `%s`, `<password>`, `xxx`, `password`, `pass`) don't count.

## Why It Matters

Hardcoded credentials:

- Get committed to version control
- Are visible to anyone with code access
- Can't be rotated without code changes
- Cause security incidents

## Examples

### Bad: Hardcoded Password

```go
const dbPassword = "super_secret_123"

const dsn = "postgres://app:super_secret_123@localhost/db"
```

### Bad: Hardcoded API Key

```go
var apiKey = "sk_live_abc123xyz789"

var authHeader = "Bearer sk_live_abc123xyz789"
```

### Good: Environment Variables

```go
func connect() {
    password := os.Getenv("DB_PASSWORD")
    if password == "" {
        log.Fatal("DB_PASSWORD not set")
    }
    db.Connect(fmt.Sprintf("user:%s@localhost/db", password))
}
```

### Good: Configuration

```go
type Config struct {
    DBPassword string `env:"DB_PASSWORD"`
    APIKey     string `env:"API_KEY"`
}

func NewService(cfg Config) *Service {
    // Credentials injected, not hardcoded
}
```

### Good: Secret Management

```go
func getSecret(ctx context.Context, name string) (string, error) {
    // Fetch from secret manager (Vault, AWS Secrets Manager, etc.)
    return secretClient.GetSecret(ctx, name)
}
```

## False Positives

The analyzer may flag:

- Test data (use environment variables in tests too)
- Example values in documentation
- Placeholder values

For test data, use constants clearly marked as fake:

```go
// testdata.go
const (
    // FakeAPIKey is for testing only - not a real key
    FakeAPIKey = "test_fake_key_not_real"
)
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  hardcodedcreds: true  # enabled by default
```

## When to Disable

Generally not recommended. If needed for test files:

```yaml
analyzers:
  hardcodedcreds: false  # Not recommended
```

## Related Analyzers

- [exporteddoc](/reference/analyzers/exporteddoc) - Documentation
