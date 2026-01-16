---
title: mockverify
permalink: /reference/analyzers/mockverify
createTime: 2025/01/16 10:00:00
---

Ensures mock implementations have compile-time interface verification.

## Category

Testability

## What It Checks

This analyzer detects mock implementations that don't verify they implement their interface at compile time.

## Why It Matters

Without compile-time verification, interface changes don't cause compilation errors in mocks. Tests pass with incomplete mocks, then fail mysteriously at runtime.

## Examples

### Bad: No Verification

```go
type MockStorage struct {
    GetFunc func(key string) (string, error)
}

func (m *MockStorage) Get(key string) (string, error) {
    return m.GetFunc(key)
}

// If Storage interface changes, this still compiles!
```

### Good: Compile-Time Verification

```go
type MockStorage struct {
    GetFunc    func(key string) (string, error)
    SetFunc    func(key, value string) error
    DeleteFunc func(key string) error
}

// Compile-time check - fails if interface changes
var _ Storage = (*MockStorage)(nil)

func (m *MockStorage) Get(key string) (string, error) {
    return m.GetFunc(key)
}

func (m *MockStorage) Set(key, value string) error {
    return m.SetFunc(key, value)
}

func (m *MockStorage) Delete(key string) error {
    return m.DeleteFunc(key)
}
```

## The Verification Pattern

```go
var _ InterfaceName = (*MockTypeName)(nil)
```

This:

1. Creates a nil pointer of the mock type
2. Assigns it to the interface type
3. Fails compilation if the mock doesn't implement the interface

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  mockverify: true  # enabled by default
```

## When to Disable

- Using mock generation tools that handle this automatically

```yaml
analyzers:
  mockverify: false
```

## Related Analyzers

- [interfaceconsistency](/reference/analyzers/interfaceconsistency) - Interface implementations
- [clockinterface](/reference/analyzers/clockinterface) - Time interface pattern
