---
title: Philosophy
permalink: /understanding/philosophy
createTime: 2025/01/16 10:00:00
---

golint-sl enforces patterns that prevent production incidents. This page explains the thinking behind the rules.

## Core Beliefs

### 1. Most Bugs Are Patterns

Production incidents rarely come from novel bugs. They come from the same patterns repeating:

- Nil pointer dereference
- Unclosed resources
- Lost error context
- Missing timeouts
- Scattered logging

If a pattern has caused one incident, it will cause another. Codify the fix as a lint rule.

### 2. Static Analysis Should Be Actionable

Every golint-sl diagnostic includes:

- What the problem is
- Why it matters
- How to fix it

Bad:

```text
nilcheck: potential nil dereference
```

Good:

```text
nilcheck: pointer parameter "user" used without nil check;
          add 'if user == nil { return ... }' at function start
```

If an analyzer can't suggest a fix, it shouldn't report.

### 3. False Positives Destroy Trust

If developers learn to ignore warnings, the linter is useless. Every diagnostic must indicate a real issue worth fixing.

We'd rather miss some bugs than cry wolf. If an analyzer produces false positives, we fix it or remove it.

### 4. Production Focus Over Style

We don't lint for:

- Tab vs. spaces
- Line length
- Comment style
- Import ordering

These are style preferences. They don't cause incidents.

We do lint for:

- Nil safety
- Resource leaks
- Error handling
- Context propagation
- Observability

These prevent 3 AM pages.

## Key Patterns

### Fail at Boundaries

Validate inputs at function boundaries, not deep in call stacks:

```go
// Bad: Panic buried deep in the code
func ProcessOrder(order *Order) error {
    items := calculateItems(order)  // Panics if order is nil!
    return saveItems(items)
}

// Good: Validate at the boundary
func ProcessOrder(order *Order) error {
    if order == nil {
        return errors.New("order cannot be nil")
    }
    items := calculateItems(order)
    return saveItems(items)
}
```

The `nilcheck` analyzer enforces this.

### Wide Events Over Scattered Logs

Based on the philosophy from [**loggingsucks.com**](https://loggingsucks.com/): one log line with 50 fields beats 15 log lines with 3 fields each.

```go
// Bad: Scattered logs
func ProcessRequest(ctx context.Context, req *Request) error {
    log.Info("starting request")
    user, err := getUser(ctx, req.UserID)
    if err != nil {
        log.Error("failed to get user", err)
        return err
    }
    log.Info("got user", user.Name)
    // ... more scattered logs ...
}

// Good: Single wide event
func ProcessRequest(ctx context.Context, req *Request) error {
    start := time.Now()
    var result string
    var userID string

    defer func() {
        logger.Info("request processed",
            zap.String("request_id", req.ID),
            zap.String("user_id", userID),
            zap.String("result", result),
            zap.Duration("duration", time.Since(start)),
            zap.String("trace_id", trace.SpanFromContext(ctx).SpanContext().TraceID().String()),
        )
    }()

    user, err := getUser(ctx, req.UserID)
    if err != nil {
        result = "user_fetch_failed"
        return err
    }
    userID = user.ID
    result = "success"
    return nil
}
```

The `wideevents` analyzer enforces this pattern based on [Logging Sucks](https://loggingsucks.com/).

### Context All The Way Down

Context carries deadlines, cancellation, and trace IDs. Dropping context breaks all three:

```go
// Bad: Context dropped
func ProcessJob(ctx context.Context, job *Job) error {
    result := heavyComputation()  // No context! Can't cancel!
    return saveResult(ctx, result)
}

// Good: Context propagated
func ProcessJob(ctx context.Context, job *Job) error {
    result, err := heavyComputation(ctx)  // Cancelable
    if err != nil {
        return err
    }
    return saveResult(ctx, result)
}
```

The `contextpropagation` analyzer ensures context flows through your entire call chain.

### Errors Need Context

Bare error returns lose the call chain:

```go
// Bad: Error loses context
func ProcessUser(id string) error {
    user, err := db.GetUser(id)
    if err != nil {
        return err  // Where did this come from?
    }
    return user.Validate()
}

// Good: Error with context
func ProcessUser(id string) error {
    user, err := db.GetUser(id)
    if err != nil {
        return fmt.Errorf("get user %s: %w", id, err)
    }
    if err := user.Validate(); err != nil {
        return fmt.Errorf("validate user %s: %w", id, err)
    }
    return nil
}
```

The `errorwrap` analyzer catches bare returns.

### Close What You Open

Unclosed resources leak memory, file handles, and connections:

```go
// Bad: Response body never closed
func fetchData(url string) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    return io.ReadAll(resp.Body)  // Leak!
}

// Good: Always close
func fetchData(url string) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}
```

The `resourceclose` analyzer catches unclosed resources.

## Why These Rules?

Every analyzer in golint-sl exists because the pattern it enforces has:

1. **Caused real incidents** - We've been paged for this
2. **Been found in code review** - Repeatedly
3. **Has a clear fix** - Not subjective

We don't add analyzers for theoretical best practices. We add them for patterns that have bitten us.

## Inspiration

golint-sl is inspired by:

- [Clean Go Code](https://github.com/Pungyeon/clean-go-article) - Variable scope, early returns, function size
- [Logging Sucks](https://loggingsucks.com/) - Wide events over scattered logs
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) - Reconciler patterns
- Production experience at SpechtLabs - Everything else

## Next Steps

- [Analyzer Categories](/understanding/categories) - How analyzers are organized
- [Wide Events Pattern](/understanding/wide-events) - Deep dive into observability
- [Kubernetes Patterns](/understanding/kubernetes-patterns) - Controller best practices
