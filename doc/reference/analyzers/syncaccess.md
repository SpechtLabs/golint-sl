---
title: syncaccess
permalink: /reference/analyzers/syncaccess
createTime: 2025/01/16 10:00:00
---

Detects potential data races and synchronization issues.

## Category

Safety

## What It Checks

This analyzer detects:

- Unsynchronized access to shared variables
- Missing mutex locks
- Potential race conditions

## Why It Matters

Data races cause:

- Corrupted data
- Unpredictable behavior
- Hard-to-reproduce bugs
- Security vulnerabilities

## Examples

### Bad: Unsynchronized Counter

```go
type Counter struct {
    value int  // Accessed from multiple goroutines
}

func (c *Counter) Increment() {
    c.value++  // Data race!
}

func (c *Counter) Get() int {
    return c.value  // Data race!
}
```

### Good: Mutex Protection

```go
type Counter struct {
    mu    sync.Mutex
    value int
}

func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.value++
}

func (c *Counter) Get() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.value
}
```

### Good: Atomic Operations

```go
type Counter struct {
    value atomic.Int64
}

func (c *Counter) Increment() {
    c.value.Add(1)
}

func (c *Counter) Get() int64 {
    return c.value.Load()
}
```

### Bad: Map Access

```go
type Cache struct {
    data map[string]string
}

func (c *Cache) Set(key, value string) {
    c.data[key] = value  // Data race!
}

func (c *Cache) Get(key string) string {
    return c.data[key]  // Data race!
}
```

### Good: sync.Map or RWMutex

```go
type Cache struct {
    data sync.Map
}

func (c *Cache) Set(key, value string) {
    c.data.Store(key, value)
}

func (c *Cache) Get(key string) (string, bool) {
    v, ok := c.data.Load(key)
    if !ok {
        return "", false
    }
    return v.(string), true
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  syncaccess: true  # enabled by default
```

## When to Disable

- Single-threaded code
- Code protected by external synchronization

```yaml
analyzers:
  syncaccess: false
```

## Related Analyzers

- [goroutineleak](/reference/analyzers/goroutineleak) - Goroutine safety
- [nilcheck](/reference/analyzers/nilcheck) - Nil safety

## See Also

- [Go Race Detector](https://go.dev/doc/articles/race_detector)
