---
title: wideevents
permalink: /reference/analyzers/wideevents
createTime: 2025/01/16 10:00:00
---

Enforces the wide events logging pattern from [**loggingsucks.com**](https://loggingsucks.com/).

## Category

Observability

## What It Checks

This analyzer enforces the philosophy from [loggingsucks.com](https://loggingsucks.com/):

1. **Banned loggers** - logrus, stdlib log, fmt.Print (use zap instead)
2. **Single event per function** - no scattered logs (consider emitting one wide event)
3. **Structured fields on log calls** - use `zap.String()`, `zap.Error()`, etc.
4. **Request context in wide events** - include trace_id, request_id, or span_id
5. **Span attributes when context is available** - use `trace.SpanFromContext(ctx)` and `span.SetAttributes()`

### Supported Logging Frameworks

- **zap** - `zap.L().Info()`, `zap.L().Error()`, etc.
- **otelzap** - `otelzap.L().InfoContext()`, `otelzap.L().WithError().ErrorContext()`, etc.

The analyzer recognizes:

- Context-aware methods (`*Context` suffix) that auto-extract trace context
- Method chaining (`.WithError()`, `.With()`) that adds structured fields

## Why It Matters

> "One log line per request per service with 50+ structured fields beats 15 scattered log statements."
>
> - [loggingsucks.com](https://loggingsucks.com/)

Scattered logs make debugging impossible at scale. See [Wide Events Pattern](/understanding/wide-events) for the full philosophy.

## Examples

### Bad: Scattered Logs

```go
func ProcessRequest(ctx context.Context, req *Request) error {
    log.Info("starting request")  // Scattered log #1

    user, err := getUser(ctx, req.UserID)
    if err != nil {
        log.Error("failed to get user", err)  // Scattered log #2
        return err
    }
    log.Info("got user")  // Scattered log #3

    return nil
}
```

### Good: Single Wide Event

```go
func ProcessRequest(ctx context.Context, req *Request) error {
    start := time.Now()
    var result string

    defer func() {
        logger.Info("request processed",
            zap.String("request_id", req.ID),
            zap.String("user_id", req.UserID),
            zap.String("result", result),
            zap.Duration("duration", time.Since(start)),
        )
    }()

    user, err := getUser(ctx, req.UserID)
    if err != nil {
        result = "user_fetch_failed"
        return err
    }

    result = "success"
    return nil
}
```

### Bad: Banned Logger

```go
import "github.com/sirupsen/logrus"

func Process() {
    logrus.Info("message")  // Banned: use zap instead
}
```

### Good: Allowed Logger

```go
import "go.uber.org/zap"

func Process() {
    zap.L().Info("message",
        zap.String("key", "value"),
    )
}
```

### Bad: Log in Loop

```go
for _, item := range items {
    logger.Info("processing item", zap.String("id", item.ID))  // Log spam!
}
```

### Good: Aggregate and Log Once

```go
var processedIDs []string
for _, item := range items {
    process(item)
    processedIDs = append(processedIDs, item.ID)
}
logger.Info("processed items", zap.Strings("ids", processedIDs))
```

### Bad: Logging Without Span Attributes

```go
func ProcessRequest(ctx context.Context, req *Request) error {
    // Has context but only logs - doesn't use span!
    logger.Info("processing request",
        zap.String("request_id", req.ID),
    )
    return nil
}
```

### Good: Using Span Attributes

```go
import "go.opentelemetry.io/otel/trace"

func ProcessRequest(ctx context.Context, req *Request) error {
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(
        attribute.String("request_id", req.ID),
        attribute.String("user_id", req.UserID),
    )

    // ... process request ...

    span.SetAttributes(attribute.String("result", "success"))
    return nil
}
```

### Best: Span Attributes + Single Wide Event

```go
func ProcessRequest(ctx context.Context, req *Request) error {
    span := trace.SpanFromContext(ctx)
    start := time.Now()

    defer func() {
        // Add final attributes to span
        span.SetAttributes(
            attribute.Int64("duration_ms", time.Since(start).Milliseconds()),
        )
        // Emit single wide event for logs/metrics
        logger.Info("request processed",
            zap.String("trace_id", span.SpanContext().TraceID().String()),
            zap.String("request_id", req.ID),
            zap.Duration("duration", time.Since(start)),
        )
    }()

    span.SetAttributes(
        attribute.String("request_id", req.ID),
        attribute.String("user_id", req.UserID),
    )

    return processInternal(ctx, req)
}
```

## Allowed Patterns

### Debug Logging

`zap.Debug` is allowed for development:

```go
zap.L().Debug("intermediate state", zap.Any("data", data))  // OK
```

### Context-Aware Methods (otelzap)

Methods ending in `Context` (like `ErrorContext`, `InfoContext`, `WarnContext`) automatically extract trace context from the `context.Context` parameter, so they don't need explicit `trace_id`/`span_id` fields:

```go
import "github.com/spechtlabs/go-otel-utils/otelzap"

func ProcessRequest(ctx context.Context, req *Request) error {
    // OK - ErrorContext automatically extracts trace_id/span_id from ctx
    otelzap.L().ErrorContext(ctx, "request failed",
        zap.String("request_id", req.ID),
    )
    return err
}
```

### Method Chaining

Fields added via method chaining (`.WithError()`, `.With()`, etc.) are recognized as structured fields:

```go
// OK - WithError adds structured error field
otelzap.L().WithError(err).ErrorContext(ctx, "operation failed")

// OK - With adds structured fields
logger.With(zap.String("component", "api")).Info("starting")
```

### Test Functions

Logging in tests is not checked:

```go
func TestProcess(t *testing.T) {
    log.Println("test output")  // OK in tests
}
```

### CLI Output

`fmt.Print*` functions are allowed in CLI packages (`cmd/` and `cli/` directories) since they're used for user output, not logging:

```go
// In cmd/mycli/main.go or internal/cli/output.go
func printResult(result string) {
    fmt.Println(result)  // OK - user output in CLI code
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  wideevents: true  # enabled by default
```

## When to Disable

- Projects using different logging patterns (e.g., controller-runtime)
- CLI tools where structured logging is overkill
- Libraries (let consumers decide logging)

```yaml
analyzers:
  wideevents: false
```

## Related Analyzers

- [contextlogger](/reference/analyzers/contextlogger) - Context-based logging
- [contextpropagation](/reference/analyzers/contextpropagation) - Context propagation

## See Also

- [Wide Events Pattern](/understanding/wide-events)
- [Logging Sucks](https://loggingsucks.com/)
