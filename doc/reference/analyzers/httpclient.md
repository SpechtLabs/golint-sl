---
title: httpclient
permalink: /reference/analyzers/httpclient
createTime: 2025/01/16 10:00:00
---

Enforces HTTP client best practices including timeouts.

## Category

Resources

## What It Checks

This analyzer detects:

- HTTP clients without timeouts
- Use of `http.DefaultClient` (has no timeout)
- Missing context in requests

## Why It Matters

HTTP requests without timeouts can hang forever:

- Slow servers cause goroutine leaks
- Connection pool exhaustion
- Service unresponsiveness

## Examples

### Bad: Default Client

```go
func fetchData(url string) ([]byte, error) {
    // http.DefaultClient has no timeout!
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}
```

### Good: Custom Client with Timeout

```go
var httpClient = &http.Client{
    Timeout: 30 * time.Second,
}

func fetchData(url string) ([]byte, error) {
    resp, err := httpClient.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}
```

### Good: Request with Context

```go
func fetchData(ctx context.Context, url string) ([]byte, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}
```

### Recommended Client Configuration

```go
var httpClient = &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        DialContext: (&net.Dialer{
            Timeout:   10 * time.Second,
            KeepAlive: 30 * time.Second,
        }).DialContext,
        MaxIdleConns:          100,
        MaxIdleConnsPerHost:   10,
        IdleConnTimeout:       90 * time.Second,
        TLSHandshakeTimeout:   10 * time.Second,
        ExpectContinueTimeout: 1 * time.Second,
    },
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  httpclient: true  # enabled by default
```

## When to Disable

- Internal tools where timeout isn't critical
- Tests with mock servers

```yaml
analyzers:
  httpclient: false
```

## Related Analyzers

- [resourceclose](/reference/analyzers/resourceclose) - Resource closing
- [contextpropagation](/reference/analyzers/contextpropagation) - Context usage
