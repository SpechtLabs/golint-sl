---
title: closurecomplexity
permalink: /reference/analyzers/closurecomplexity
createTime: 2025/01/16 10:00:00
---

Detects complex closures that should be extracted to named functions.

## Category

Clean Code

## What It Checks

This analyzer detects anonymous functions (closures) that are too complex.

**Thresholds:**

- Maximum statements: 15
- Maximum nesting depth: 2
- Maximum captured variables: 5

**Exempt Closures:**

- Deferred closures (`defer func() {...}()`)
- Goroutine closures (`go func() {...}()`)
- Closures returned from functions (handler factory pattern)
- Cobra command handlers (`RunE`, `Run`, `PreRunE`, etc.)
- HTTP handler fields
- Visitor pattern callbacks (`Inspect`, `VisitAll`, `Walk`, `WalkDir`, etc.)
- Test files (closures in `*_test.go` files)

## Why It Matters

Complex closures:

- Are hard to test in isolation
- Reduce code readability
- Hide important logic
- Make debugging difficult

## Examples

### Bad: Complex Closure

```go
func ProcessItems(items []Item) error {
    return withTransaction(func(tx *sql.Tx) error {
        // 50+ lines of complex logic
        for _, item := range items {
            if item.Type == "special" {
                result, err := tx.Query("SELECT ...")
                if err != nil {
                    return err
                }
                // ... more complex processing ...
                for result.Next() {
                    // ... even more logic ...
                }
            }
        }
        return nil
    })
}
```

### Good: Extract to Named Function

```go
func ProcessItems(items []Item) error {
    return withTransaction(func(tx *sql.Tx) error {
        return processItemsInTx(tx, items)
    })
}

func processItemsInTx(tx *sql.Tx, items []Item) error {
    for _, item := range items {
        if err := processSpecialItem(tx, item); err != nil {
            return err
        }
    }
    return nil
}

func processSpecialItem(tx *sql.Tx, item Item) error {
    if item.Type != "special" {
        return nil
    }
    // Clear, testable logic
    result, err := tx.Query("SELECT ...")
    if err != nil {
        return err
    }
    defer result.Close()
    // ...
    return nil
}
```

### Acceptable: Simple Closures

```go
// Simple closures are fine
sort.Slice(items, func(i, j int) bool {
    return items[i].Name < items[j].Name
})

// Short handlers are fine
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
})
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  closurecomplexity: true  # enabled by default
```

## When to Disable

- Code with many simple callbacks
- Generated code

```yaml
analyzers:
  closurecomplexity: false
```

## Related Analyzers

- [functionsize](/reference/analyzers/functionsize) - Function length
- [nestingdepth](/reference/analyzers/nestingdepth) - Nesting limits
