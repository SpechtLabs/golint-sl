---
title: resourceclose
permalink: /reference/analyzers/resourceclose
createTime: 2025/01/16 10:00:00
---

Detects unclosed resources like HTTP response bodies and files.

## Category

Resources

## What It Checks

This analyzer detects resources that are opened but not closed:

- HTTP response bodies (`resp.Body`)
- Files (`os.Open`, `os.Create`)
- Database connections
- Network connections

## Why It Matters

Unclosed resources cause:

- Connection pool exhaustion
- File descriptor leaks
- Memory leaks
- Eventually, service failure

## Examples

### Bad: Response Body Not Closed

```go
func fetchData(url string) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    // Body never closed - connection leaks!
    return io.ReadAll(resp.Body)
}
```

### Good: Response Body Closed

```go
func fetchData(url string) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}
```

### Bad: File Not Closed

```go
func readConfig(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    // File never closed!
    return io.ReadAll(f)
}
```

### Good: File Closed

```go
func readConfig(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    return io.ReadAll(f)
}
```

### Good: Using os.ReadFile

```go
func readConfig(path string) ([]byte, error) {
    return os.ReadFile(path)  // Handles close internally
}
```

## The Defer Pattern

Always close resources with `defer` immediately after opening:

```go
resource, err := openResource()
if err != nil {
    return err
}
defer resource.Close()  // Immediately after error check
// Use resource...
```

### Deferred Anonymous Functions

The analyzer also detects close calls inside deferred anonymous functions:

```go
resp, err := http.Get(url)
if err != nil {
    return nil, err
}
defer func() { _ = resp.Body.Close() }()  // Also detected
```

### Test Cleanup

In tests, `t.Cleanup()` is recognized as a valid close pattern:

```go
func TestFetch(t *testing.T) {
    f, err := os.CreateTemp("", "test")
    require.NoError(t, err)
    t.Cleanup(func() { _ = f.Close() })  // Recognized as close
}
```

## Excluded Resources

Standard streams (`os.Stdout`, `os.Stderr`, `os.Stdin`) are excluded - these should never be closed by user code:

```go
func printOutput() {
    output := os.Stdout  // Not flagged - shouldn't close stdout
    fmt.Fprintln(output, "message")
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  resourceclose: true  # enabled by default
```

## When to Disable

This analyzer should rarely be disabled. Resource leaks are serious bugs.

```yaml
analyzers:
  resourceclose: false  # Not recommended
```

## Related Analyzers

- [httpclient](/reference/analyzers/httpclient) - HTTP client practices
- [goroutineleak](/reference/analyzers/goroutineleak) - Goroutine leaks
