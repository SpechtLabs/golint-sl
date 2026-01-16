---
title: contextpropagation
permalink: /reference/analyzers/contextpropagation
createTime: 2025/01/16 10:00:00
---

Ensures context is propagated through all function calls.

## Category

Observability

## What It Checks

This analyzer detects functions that receive a context but don't pass it to callees that need it.

## Why It Matters

Context carries:

- **Cancellation signals**: Stop work when the request is cancelled
- **Deadlines**: Fail fast when time runs out
- **Trace IDs**: Correlate logs and spans across services
- **Request values**: User ID, request ID, etc.

Dropping context breaks all of these.

## Examples

### Bad

```go
func ProcessOrder(ctx context.Context, order *Order) error {
    // Context not passed - can't cancel this!
    user, err := fetchUser(order.UserID)
    if err != nil {
        return err
    }

    // Context not passed - deadlines ignored!
    if err := chargePayment(order.Total); err != nil {
        return err
    }

    return nil
}
```

### Good

```go
func ProcessOrder(ctx context.Context, order *Order) error {
    // Context propagated - cancelable and traceable
    user, err := fetchUser(ctx, order.UserID)
    if err != nil {
        return err
    }

    if err := chargePayment(ctx, order.Total); err != nil {
        return err
    }

    return nil
}
```

### Bad: Background Context

```go
func ProcessOrder(ctx context.Context, order *Order) error {
    // Using background context loses the parent's deadline/cancellation!
    user, err := fetchUser(context.Background(), order.UserID)
    return err
}
```

### Good: Derived Context

```go
func ProcessOrder(ctx context.Context, order *Order) error {
    // If you need a different deadline, derive from parent
    childCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    user, err := fetchUser(childCtx, order.UserID)
    return err
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  contextpropagation: true  # enabled by default
```

## When to Disable

- CLI tools that don't use context heavily
- Simple scripts without cancellation needs

```yaml
analyzers:
  contextpropagation: false
```

## Related Analyzers

- [contextfirst](/reference/analyzers/contextfirst) - Context parameter ordering
- [contextlogger](/reference/analyzers/contextlogger) - Context-based logging

## See Also

- [Go Blog: Context](https://go.dev/blog/context)
