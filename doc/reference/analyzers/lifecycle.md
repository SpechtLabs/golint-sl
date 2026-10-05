---
title: lifecycle
permalink: /reference/analyzers/lifecycle
createTime: 2025/01/16 10:00:00
---

Enforces component lifecycle patterns (Run/Close).

## Category

Architecture

## What It Checks

This analyzer ensures components that start background work have proper lifecycle methods. For every type with a run method (`Run`, `Start` or `Serve`, on a plain or generic receiver) it checks that:

- the type also has a stop method: `Close`, `Stop`, `Shutdown`, `GracefulStop` or `GracefulShutdown`
- the run method takes a `context.Context`, or a type that implements it, as its first parameter
- a run method with a long-running loop, `for {}` without a condition or a `range` over a channel, observes cancellation somewhere in its body: a receive from or a call to `ctx.Done()` (or another `Done()` channel), or a `ctx.Err()` check. Loops over slices, maps and integers and loops with a condition are bounded and need no check.

## Why It Matters

Components without lifecycle management:

- Leak goroutines
- Don't handle shutdown gracefully
- Can't be properly tested
- Cause resource leaks

## Examples

### Bad: No Lifecycle

```go
type Worker struct {
    done chan struct{}
}

func NewWorker() *Worker {
    w := &Worker{done: make(chan struct{})}
    go w.loop()  // Starts goroutine in constructor!
    return w
}

func (w *Worker) loop() {
    for {
        // No way to stop this!
        doWork()
    }
}
```

### Good: Proper Lifecycle

```go
type Worker struct {
    done chan struct{}
}

func NewWorker() *Worker {
    return &Worker{
        done: make(chan struct{}),
    }
}

// Run starts the worker. It blocks until ctx is cancelled or Close is called.
func (w *Worker) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-w.done:
            return nil
        default:
            doWork()
        }
    }
}

// Close stops the worker gracefully.
func (w *Worker) Close() error {
    close(w.done)
    return nil
}
```

### Usage Pattern

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    worker := NewWorker()

    // Handle shutdown
    go func() {
        sigCh := make(chan os.Signal, 1)
        signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
        <-sigCh
        worker.Close()
    }()

    if err := worker.Run(ctx); err != nil {
        log.Fatal(err)
    }
}
```

### The Lifecycle Interface

```go
type Lifecycle interface {
    Run(ctx context.Context) error
    Close() error
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  lifecycle: true  # enabled by default
```

## When to Disable

- Simple utilities without background work
- Stateless functions

```yaml
analyzers:
  lifecycle: false
```

## Related Analyzers

- [goroutineleak](/reference/analyzers/goroutineleak) - Goroutine safety
- [contextpropagation](/reference/analyzers/contextpropagation) - Context usage
