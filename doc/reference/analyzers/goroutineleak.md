---
title: goroutineleak
permalink: /reference/analyzers/goroutineleak
createTime: 2025/01/16 10:00:00
---

Detects goroutines that may never terminate.

## Category

Safety

## What It Checks

This analyzer detects goroutines that:

- Block forever on channels
- Have no exit condition
- Ignore context cancellation

## Why It Matters

Leaked goroutines:

- Consume memory indefinitely
- Hold references preventing GC
- Eventually exhaust resources

## Examples

### Bad: Unbounded Goroutine

```go
func StartWorker() {
    go func() {
        for {
            doWork()  // Never exits!
        }
    }()
}
```

### Good: Context-Aware Goroutine

```go
func StartWorker(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return  // Clean exit
            default:
                doWork()
            }
        }
    }()
}
```

### Bad: Blocking Channel

```go
func Process(items []Item) {
    results := make(chan Result)

    for _, item := range items {
        go func(item Item) {
            results <- process(item)  // Blocks if no receiver!
        }(item)
    }

    // If we only read some results, goroutines leak
    return <-results
}
```

### Good: Buffered Channel

```go
func Process(items []Item) []Result {
    results := make(chan Result, len(items))

    for _, item := range items {
        go func(item Item) {
            results <- process(item)
        }(item)
    }

    var out []Result
    for range items {
        out = append(out, <-results)
    }
    return out
}
```

### Good: WaitGroup Pattern

```go
func Process(ctx context.Context, items []Item) {
    var wg sync.WaitGroup

    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()
            processWithContext(ctx, item)
        }(item)
    }

    wg.Wait()
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  goroutineleak: true  # enabled by default
```

## When to Disable

- Simple scripts without long-running goroutines

```yaml
analyzers:
  goroutineleak: false
```

## Related Analyzers

- [contextpropagation](/reference/analyzers/contextpropagation) - Context usage
- [resourceclose](/reference/analyzers/resourceclose) - Resource management
