---
title: contextfirst
permalink: /reference/analyzers/contextfirst
createTime: 2025/01/16 10:00:00
---

Ensures `context.Context` is the first parameter in function signatures.

## Category

Architecture

## What It Checks

This analyzer detects functions where `context.Context` is not the first parameter.

## Why It Matters

Go convention: context is always first. This:

- Makes code consistent and predictable
- Allows easy visual scanning for context usage
- Follows standard library conventions

## Examples

### Bad

```go
func GetUser(id string, ctx context.Context) (*User, error) {
    // Context buried in parameters
}

func ProcessOrder(order *Order, ctx context.Context, opts ...Option) error {
    // Context in the middle
}
```

### Good

```go
func GetUser(ctx context.Context, id string) (*User, error) {
    // Context is first
}

func ProcessOrder(ctx context.Context, order *Order, opts ...Option) error {
    // Context is first, variadic opts are last
}
```

### Parameter Order Convention

```go
func DoSomething(
    ctx context.Context,     // 1. Context first
    requiredParam Type,      // 2. Required parameters
    optionalParam *Type,     // 3. Optional parameters
    opts ...Option,          // 4. Variadic options last
) error
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  contextfirst: true  # enabled by default
```

## When to Disable

- Interface implementations that must match external signatures
- CGO or interop code

```yaml
analyzers:
  contextfirst: false
```

## Related Analyzers

- [contextpropagation](/reference/analyzers/contextpropagation) - Context propagation
- [contextlogger](/reference/analyzers/contextlogger) - Context-based logging

## See Also

- [Go Blog: Context](https://go.dev/blog/context)
