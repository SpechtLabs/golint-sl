---
title: hardcodedcreds
permalink: /reference/analyzers/hardcodedcreds
createTime: 2025/01/16 10:00:00
---

Detects potential hardcoded secrets and credentials.

## Category

Architecture

## What It Checks

This analyzer detects patterns that look like hardcoded:

- Passwords
- API keys
- Tokens
- Connection strings with credentials

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

func connect() {
    db.Connect("user:super_secret_123@localhost/db")
}
```

### Bad: Hardcoded API Key

```go
var apiKey = "sk_live_abc123xyz789"

func callAPI() {
    req.Header.Set("Authorization", "Bearer sk_live_abc123xyz789")
}
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
