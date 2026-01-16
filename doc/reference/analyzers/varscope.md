---
title: varscope
permalink: /reference/analyzers/varscope
createTime: 2025/01/16 10:00:00
---

Ensures variables are declared close to their usage.

## Category

Clean Code

## What It Checks

This analyzer detects variables declared far from where they're used.

## Why It Matters

Variables declared far from usage:

- Force readers to scroll and remember
- Increase cognitive load
- Make refactoring harder

## Examples

### Bad: Variable Far From Usage

```go
func Process(items []Item) int {
    var total int  // Declared here...

    // 50 lines of unrelated code
    validate(items)
    filter(items)
    transform(items)
    // ... more code ...

    // ...used here
    for _, item := range items {
        total += item.Value
    }
    return total
}
```

### Good: Variable Near Usage

```go
func Process(items []Item) int {
    // Validation and transformation
    validate(items)
    filter(items)
    transform(items)

    // Variable declared right before use
    var total int
    for _, item := range items {
        total += item.Value
    }
    return total
}
```

### Good: Inline Declaration

```go
func Process(items []Item) int {
    validate(items)
    filter(items)
    transform(items)

    // Even better: declare in the statement
    total := 0
    for _, item := range items {
        total += item.Value
    }
    return total
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  varscope: true  # enabled by default
```

## When to Disable

- Performance-critical code where allocation matters
- Complex algorithms where variable declarations at the top aid understanding

```yaml
analyzers:
  varscope: false
```

## Related Analyzers

- [nestingdepth](/reference/analyzers/nestingdepth) - Code structure
- [functionsize](/reference/analyzers/functionsize) - Function length
