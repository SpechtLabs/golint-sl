---
title: contextlogger
permalink: /reference/analyzers/contextlogger
createTime: 2025/01/16 10:00:00
---

Enforces context-based logging patterns.

## Category

Observability

## What It Checks

This analyzer ensures loggers use context to include request-scoped information like trace IDs and request IDs.

## Why It Matters

Logs without context can't be correlated:

```text
INFO: user created          <- Which request?
INFO: order processed       <- Same request? Different request?
```

With context:

```text
INFO: user created    request_id=abc123 trace_id=xyz789
INFO: order processed request_id=abc123 trace_id=xyz789  <- Same request!
```

## Examples

### Bad

```go
func ProcessRequest(ctx context.Context, req *Request) error {
    logger.Info("processing")  // No context!
    return nil
}
```

### Good

```go
func ProcessRequest(ctx context.Context, req *Request) error {
    // Get logger from context (includes trace_id, request_id)
    logger := logging.FromContext(ctx)
    logger.Info("processing")
    return nil
}
```

### Pattern: Logger in Context

```go
// middleware.go
func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := uuid.New().String()
        traceID := trace.SpanFromContext(r.Context()).SpanContext().TraceID().String()

        logger := zap.L().With(
            zap.String("request_id", requestID),
            zap.String("trace_id", traceID),
        )

        ctx := logging.WithLogger(r.Context(), logger)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// logging/context.go
type ctxKey struct{}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
    return context.WithValue(ctx, ctxKey{}, logger)
}

func FromContext(ctx context.Context) *zap.Logger {
    if logger, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
        return logger
    }
    return zap.L()  // Fallback to global logger
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  contextlogger: true  # enabled by default
```

## When to Disable

- Using controller-runtime logging (has its own context patterns)
- CLI applications without request context
- Libraries (let consumers decide logging)

```yaml
analyzers:
  contextlogger: false
```

## Related Analyzers

- [wideevents](/reference/analyzers/wideevents) - Wide event logging
- [contextpropagation](/reference/analyzers/contextpropagation) - Context propagation
