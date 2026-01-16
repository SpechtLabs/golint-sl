---
title: clockinterface
permalink: /reference/analyzers/clockinterface
createTime: 2025/01/16 10:00:00
---

Enforces using a Clock interface for testable time operations.

## Category

Testability

## What It Checks

This analyzer detects direct calls to `time.Now()` that should use an injectable Clock interface.

## Why It Matters

Direct time calls are untestable:

```go
func IsExpired(token *Token) bool {
    return time.Now().After(token.ExpiresAt)  // Can't test!
}
```

With a Clock interface, you can inject test time:

```go
func (s *Service) IsExpired(token *Token) bool {
    return s.clock.Now().After(token.ExpiresAt)  // Testable!
}
```

## Examples

### Bad

```go
type Service struct {
    db *sql.DB
}

func (s *Service) CreateToken() *Token {
    return &Token{
        CreatedAt: time.Now(),  // Hardcoded time
        ExpiresAt: time.Now().Add(24 * time.Hour),
    }
}
```

### Good

```go
type Clock interface {
    Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time {
    return time.Now()
}

type Service struct {
    db    *sql.DB
    clock Clock
}

func (s *Service) CreateToken() *Token {
    now := s.clock.Now()
    return &Token{
        CreatedAt: now,
        ExpiresAt: now.Add(24 * time.Hour),
    }
}
```

### Test with Mock Clock

```go
type MockClock struct {
    current time.Time
}

func (m *MockClock) Now() time.Time {
    return m.current
}

func (m *MockClock) Advance(d time.Duration) {
    m.current = m.current.Add(d)
}

func TestTokenExpiry(t *testing.T) {
    clock := &MockClock{current: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
    svc := &Service{clock: clock}

    token := svc.CreateToken()

    // Advance time past expiry
    clock.Advance(25 * time.Hour)

    if !svc.IsExpired(token) {
        t.Error("token should be expired")
    }
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  clockinterface: true  # enabled by default
```

## When to Disable

- Simple scripts without tests
- Code where time testing isn't needed

```yaml
analyzers:
  clockinterface: false
```

## Related Analyzers

- [interfaceconsistency](/reference/analyzers/interfaceconsistency) - Interface patterns
- [mockverify](/reference/analyzers/mockverify) - Mock verification
