---
title: interfaceconsistency
permalink: /reference/analyzers/interfaceconsistency
createTime: 2025/01/16 10:00:00
---

Ensures interface implementations are complete and consistent.

## Category

Testability

## What It Checks

This analyzer detects incomplete or inconsistent interface implementations.

## Why It Matters

Incomplete implementations cause runtime errors or unexpected behavior. Catching them at lint time prevents production issues.

## Examples

### Bad: Incomplete Implementation

```go
type Storage interface {
    Get(key string) (string, error)
    Set(key string, value string) error
    Delete(key string) error
}

type MemoryStorage struct {
    data map[string]string
}

func (m *MemoryStorage) Get(key string) (string, error) {
    return m.data[key], nil
}

func (m *MemoryStorage) Set(key string, value string) error {
    m.data[key] = value
    return nil
}

// Delete is missing!
```

### Good: Complete Implementation

```go
type MemoryStorage struct {
    data map[string]string
}

func (m *MemoryStorage) Get(key string) (string, error) {
    return m.data[key], nil
}

func (m *MemoryStorage) Set(key string, value string) error {
    m.data[key] = value
    return nil
}

func (m *MemoryStorage) Delete(key string) error {
    delete(m.data, key)
    return nil
}

// Compile-time verification
var _ Storage = (*MemoryStorage)(nil)
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  interfaceconsistency: true  # enabled by default
```

## When to Disable

- Projects with minimal interface usage

```yaml
analyzers:
  interfaceconsistency: false
```

## Related Analyzers

- [mockverify](/reference/analyzers/mockverify) - Mock verification
- [returninterface](/reference/analyzers/returninterface) - Return type patterns
