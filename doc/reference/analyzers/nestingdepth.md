---
title: nestingdepth
permalink: /reference/analyzers/nestingdepth
createTime: 2025/01/16 10:00:00
---

Enforces shallow nesting with early returns.

## Category

Safety

## What It Checks

This analyzer detects deeply nested code that should use early returns instead.

## Why It Matters

Deep nesting is hard to read and reason about:

```go
func Process(x int) error {
    if x > 0 {
        if x < 100 {
            if isValid(x) {
                if hasPermission() {
                    return doWork(x)  // Where am I?
                }
            }
        }
    }
    return nil
}
```

## Examples

### Bad: Deep Nesting

```go
func Process(user *User) error {
    if user != nil {
        if user.Active {
            if user.HasPermission("write") {
                if user.Quota > 0 {
                    return performAction(user)
                } else {
                    return ErrQuotaExceeded
                }
            } else {
                return ErrNoPermission
            }
        } else {
            return ErrUserInactive
        }
    } else {
        return ErrNilUser
    }
}
```

### Good: Early Returns

```go
func Process(user *User) error {
    if user == nil {
        return ErrNilUser
    }
    if !user.Active {
        return ErrUserInactive
    }
    if !user.HasPermission("write") {
        return ErrNoPermission
    }
    if user.Quota <= 0 {
        return ErrQuotaExceeded
    }
    return performAction(user)
}
```

### The Pattern

1. Check error conditions first
2. Return early on failure
3. Happy path flows straight down

```go
func Process(input Input) (Output, error) {
    // Validation - early returns
    if input.A == "" {
        return Output{}, errors.New("A is required")
    }
    if input.B < 0 {
        return Output{}, errors.New("B must be positive")
    }

    // Happy path - no nesting
    result := compute(input.A, input.B)
    return Output{Value: result}, nil
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  nestingdepth: true  # enabled by default
```

## When to Disable

- Complex algorithms where nesting is unavoidable
- Generated code

```yaml
analyzers:
  nestingdepth: false
```

## Related Analyzers

- [functionsize](/reference/analyzers/functionsize) - Function complexity
- [varscope](/reference/analyzers/varscope) - Variable scope
