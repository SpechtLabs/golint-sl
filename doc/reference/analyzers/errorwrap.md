---
title: errorwrap
permalink: /reference/analyzers/errorwrap
createTime: 2025/01/16 10:00:00
---

Detects bare error returns that lose context.

## Category

Error Handling

## What It Checks

This analyzer finds error returns that don't add context, making debugging difficult.

## Why It Matters

Bare error returns lose the call chain:

```text
Error: connection refused
```

With context, you can trace the error:

```text
Error: get user "alice": fetch from database: connection refused
```

## Examples

### Bad

```go
func ProcessOrder(orderID string) error {
    order, err := db.GetOrder(orderID)
    if err != nil {
        return err  // Lost context: what were we doing?
    }

    if err := validateOrder(order); err != nil {
        return err  // Lost context: which order failed?
    }

    return nil
}
```

### Good

```go
func ProcessOrder(orderID string) error {
    order, err := db.GetOrder(orderID)
    if err != nil {
        return fmt.Errorf("get order %s: %w", orderID, err)
    }

    if err := validateOrder(order); err != nil {
        return fmt.Errorf("validate order %s: %w", orderID, err)
    }

    return nil
}
```

## The %w Verb

Use `%w` (not `%v` or `%s`) to wrap errors:

```go
// Good: preserves error chain for errors.Is/As
return fmt.Errorf("context: %w", err)

// Bad: breaks error chain
return fmt.Errorf("context: %v", err)
```

## Exceptions

The analyzer allows bare returns in certain cases:

```go
// Allowed: returning sentinel errors
if notFound {
    return ErrNotFound  // Sentinel error, context not needed
}

// Allowed: simple getters that add no context
func (s *Service) Client() *http.Client {
    return s.client
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  errorwrap: true  # enabled by default
```

## When to Disable

- Very simple functions where context is obvious
- Performance-critical paths (wrapping has overhead)

```yaml
analyzers:
  errorwrap: false
```

## Related Analyzers

- [humaneerror](/reference/analyzers/humaneerror) - User-facing errors
- [sentinelerrors](/reference/analyzers/sentinelerrors) - Sentinel error patterns

## See Also

- [Go Blog: Working with Errors](https://go.dev/blog/go1.13-errors)
