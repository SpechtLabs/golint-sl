---
title: emptyinterface
permalink: /reference/analyzers/emptyinterface
createTime: 2025/01/16 10:00:00
---

Flags problematic usage of `interface{}` or `any`.

## Category

Clean Code

## What It Checks

This analyzer detects uses of empty interface (`interface{}` or `any`) that could be replaced with concrete types or proper interfaces.

## Why It Matters

Empty interface:

- Bypasses type safety
- Requires type assertions at runtime
- Makes code harder to understand
- Hides API contracts

## Examples

### Bad: Empty Interface Parameter

```go
func Process(data interface{}) error {
    // What types are valid? Unknown!
    switch v := data.(type) {
    case string:
        return processString(v)
    case int:
        return processInt(v)
    default:
        return errors.New("unsupported type")  // Runtime error!
    }
}
```

### Good: Specific Interface

```go
type Processable interface {
    Process() error
}

func Process(data Processable) error {
    return data.Process()  // Type-safe!
}
```

### Good: Generics (Go 1.18+)

```go
func Process[T Processable](data T) error {
    return data.Process()
}
```

### Acceptable: JSON/Reflection

```go
// JSON unmarshaling needs interface{}
func parseJSON(data []byte) (map[string]interface{}, error) {
    var result map[string]interface{}
    err := json.Unmarshal(data, &result)
    return result, err
}
```

### Acceptable: Logging

```go
// Logging accepts any value
logger.Info("event", zap.Any("data", complexStruct))
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  emptyinterface: true  # enabled by default
```

## When to Disable

- Heavy use of reflection
- JSON processing code
- Plugin systems

```yaml
analyzers:
  emptyinterface: false
```

## Related Analyzers

- [returninterface](/reference/analyzers/returninterface) - Return type patterns
- [interfaceconsistency](/reference/analyzers/interfaceconsistency) - Interface implementations
