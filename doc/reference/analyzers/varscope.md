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

**Threshold:** Variables declared more than 15 lines before first use are flagged.

**Exempt Variable Names:**

Common setup/configuration variable names are exempt from the distance check:

- Configuration: `options`, `opts`, `config`, `cfg`
- Context: `ctx`, `span`
- Output: `result`, `results`, `output`, `formatted`, `content`
- Buffers: `builder`, `buf`, `buffer`
- Observability: `traceProvider`, `logProvider`, `cleanup`, `shutdown`

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

## Allowed Patterns

### Table-Driven Tests

Variables commonly used in table-driven tests are allowed at the start of test functions:

```go
func TestProcess(t *testing.T) {
    tests := []struct {  // OK - standard table-driven test pattern
        name     string
        input    int
        expected int
    }{
        {"positive", 5, 10},
        {"negative", -3, -6},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

Allowed variable names: `tests`, `testCases`, `cases`, `tt`, `tc`, `tcs`, `scenarios`.

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
