---
title: nopanic
permalink: /reference/analyzers/nopanic
createTime: 2025/01/16 10:00:00
---

Ensures library code returns errors instead of panicking.

## Category

Safety

## What It Checks

This analyzer detects `panic()` calls in library code that should return errors instead.

## Why It Matters

Libraries that panic crash their callers:

```go
// Caller's code crashes unexpectedly
result := yourlib.Process(data)  // panic!
```

Libraries should return errors, letting callers decide how to handle them:

```go
result, err := yourlib.Process(data)
if err != nil {
    // Caller handles it appropriately
}
```

## Examples

### Bad: Library Panics

```go
// mylib/processor.go
func Process(data []byte) Result {
    if len(data) == 0 {
        panic("data cannot be empty")  // Crashes caller!
    }
    // ...
}
```

### Good: Library Returns Error

```go
// mylib/processor.go
func Process(data []byte) (Result, error) {
    if len(data) == 0 {
        return Result{}, errors.New("data cannot be empty")
    }
    // ...
}
```

### Allowed: Panic in main/init

```go
// cmd/myapp/main.go
func main() {
    if err := run(); err != nil {
        panic(err)  // OK in main
    }
}

func init() {
    if os.Getenv("REQUIRED") == "" {
        panic("REQUIRED env var not set")  // OK in init
    }
}
```

### Allowed: Unreachable Code

```go
func processType(t Type) string {
    switch t {
    case TypeA:
        return "a"
    case TypeB:
        return "b"
    default:
        panic("unreachable")  // OK - indicates bug in caller
    }
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  nopanic: true  # enabled by default
```

## When to Disable

- Application code (not a library)
- Internal packages not meant for external use

```yaml
analyzers:
  nopanic: false
```

## Related Analyzers

- [nilcheck](/reference/analyzers/nilcheck) - Nil pointer safety
- [errorwrap](/reference/analyzers/errorwrap) - Error handling
