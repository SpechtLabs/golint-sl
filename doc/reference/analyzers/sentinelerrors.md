---
title: sentinelerrors
permalink: /reference/analyzers/sentinelerrors
createTime: 2025/01/16 10:00:00
---

Prefers sentinel errors over inline `errors.New()`.

## Category

Error Handling

## What It Checks

This analyzer detects inline error creation that should be sentinel errors.

## Why It Matters

Inline errors can't be checked programmatically:

```go
// Callers can't check for this specific error
if err := doThing(); err != nil {
    return errors.New("not found")  // New error every time
}

// Caller code:
if err.Error() == "not found" {  // Fragile string comparison!
    // handle not found
}
```

Sentinel errors enable proper error handling:

```go
var ErrNotFound = errors.New("not found")

func doThing() error {
    return ErrNotFound  // Same error instance
}

// Caller code:
if errors.Is(err, ErrNotFound) {  // Robust check
    // handle not found
}
```

## Examples

### Bad

```go
func GetUser(id string) (*User, error) {
    user := db.Find(id)
    if user == nil {
        return nil, errors.New("user not found")  // Inline error
    }
    return user, nil
}

func GetOrder(id string) (*Order, error) {
    order := db.Find(id)
    if order == nil {
        return nil, errors.New("order not found")  // Different message, same concept
    }
    return order, nil
}
```

### Good

```go
// Define sentinel errors at package level
var (
    ErrUserNotFound  = errors.New("user not found")
    ErrOrderNotFound = errors.New("order not found")
)

func GetUser(id string) (*User, error) {
    user := db.Find(id)
    if user == nil {
        return nil, ErrUserNotFound
    }
    return user, nil
}

func GetOrder(id string) (*Order, error) {
    order := db.Find(id)
    if order == nil {
        return nil, ErrOrderNotFound
    }
    return order, nil
}
```

## Naming Convention

Sentinel errors should:

- Be exported (start with capital letter)
- Start with `Err`
- Be descriptive

```go
var (
    ErrNotFound       = errors.New("not found")
    ErrUnauthorized   = errors.New("unauthorized")
    ErrInvalidInput   = errors.New("invalid input")
    ErrAlreadyExists  = errors.New("already exists")
)
```

## Wrapping Sentinels

You can add context while preserving the sentinel:

```go
if user == nil {
    return nil, fmt.Errorf("get user %s: %w", id, ErrNotFound)
}

// Caller can still check:
if errors.Is(err, ErrNotFound) {
    // handle
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  sentinelerrors: true  # enabled by default
```

## When to Disable

- One-off errors that are never checked
- Errors with dynamic content that can't be sentinels

```yaml
analyzers:
  sentinelerrors: false
```

## Related Analyzers

- [errorwrap](/reference/analyzers/errorwrap) - Error context wrapping
- [humaneerror](/reference/analyzers/humaneerror) - User-facing errors

## See Also

- [Go Blog: Working with Errors](https://go.dev/blog/go1.13-errors)
