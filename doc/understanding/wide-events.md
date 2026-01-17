---
title: Wide Events Pattern
permalink: /understanding/wide-events
createTime: 2025/01/16 10:00:00
---

::: tip This Pattern is Based on loggingsucks.com
The wide events pattern implemented by golint-sl is directly inspired by [**loggingsucks.com**](https://loggingsucks.com/) - a manifesto for better observability. If you haven't read it yet, stop and read it now. It will change how you think about logging.

**The core insight**: One log line per request per service with 50+ structured fields beats 15 scattered log statements with 3 fields each.
:::

The wide events pattern is a logging philosophy that dramatically improves debuggability. golint-sl's `wideevents` analyzer enforces this pattern.

## The Problem with Traditional Logging

Traditional logging scatters information across many log lines:

```go
func ProcessOrder(ctx context.Context, order *Order) error {
    log.Info("starting order processing")

    user, err := getUser(ctx, order.UserID)
    if err != nil {
        log.Error("failed to get user", "error", err)
        return err
    }
    log.Info("fetched user", "user_id", user.ID)

    inventory, err := checkInventory(ctx, order.Items)
    if err != nil {
        log.Error("inventory check failed", "error", err)
        return err
    }
    log.Info("inventory checked", "available", inventory.Available)

    payment, err := processPayment(ctx, order.Total)
    if err != nil {
        log.Error("payment failed", "error", err)
        return err
    }
    log.Info("payment processed", "transaction_id", payment.ID)

    log.Info("order completed")
    return nil
}
```

This creates several problems:

1. **Log Spam**: 15 log lines for one request
2. **Scattered Context**: Information spread across lines
3. **Hard to Correlate**: Which logs go together?
4. **Missing Context**: Each log only has partial information
5. **Expensive**: More logs = more storage = more cost

## The Wide Events Solution

Instead of many narrow logs, emit one wide event with all context:

```go
func ProcessOrder(ctx context.Context, order *Order) error {
    start := time.Now()

    // Collect all context as we go
    event := &OrderEvent{
        OrderID:   order.ID,
        UserID:    order.UserID,
        RequestID: middleware.GetRequestID(ctx),
        TraceID:   trace.SpanFromContext(ctx).SpanContext().TraceID().String(),
    }

    // Always emit the event, even on failure
    defer func() {
        event.Duration = time.Since(start)
        logger.Info("order processed",
            zap.String("order_id", event.OrderID),
            zap.String("user_id", event.UserID),
            zap.String("request_id", event.RequestID),
            zap.String("trace_id", event.TraceID),
            zap.String("result", event.Result),
            zap.String("failure_reason", event.FailureReason),
            zap.Duration("duration", event.Duration),
            zap.Int("items_count", event.ItemsCount),
            zap.Float64("total_amount", event.TotalAmount),
            zap.String("payment_method", event.PaymentMethod),
            zap.String("transaction_id", event.TransactionID),
            zap.Bool("inventory_reserved", event.InventoryReserved),
        )
    }()

    user, err := getUser(ctx, order.UserID)
    if err != nil {
        event.Result = "failed"
        event.FailureReason = "user_fetch_failed"
        return fmt.Errorf("get user: %w", err)
    }

    event.ItemsCount = len(order.Items)
    inventory, err := checkInventory(ctx, order.Items)
    if err != nil {
        event.Result = "failed"
        event.FailureReason = "inventory_check_failed"
        return fmt.Errorf("check inventory: %w", err)
    }
    event.InventoryReserved = true

    event.TotalAmount = order.Total
    event.PaymentMethod = order.PaymentMethod
    payment, err := processPayment(ctx, order.Total)
    if err != nil {
        event.Result = "failed"
        event.FailureReason = "payment_failed"
        return fmt.Errorf("process payment: %w", err)
    }
    event.TransactionID = payment.ID

    event.Result = "success"
    return nil
}
```

## Benefits

### 1. One Line Per Request

Instead of searching through 15 log lines, you have one line with everything:

```json
{
  "level": "info",
  "msg": "order processed",
  "order_id": "ord_123",
  "user_id": "usr_456",
  "request_id": "req_789",
  "trace_id": "abc123",
  "result": "success",
  "duration": "234ms",
  "items_count": 3,
  "total_amount": 99.99,
  "payment_method": "credit_card",
  "transaction_id": "txn_xyz"
}
```

### 2. Easy Debugging

When something fails:

```json
{
  "level": "info",
  "msg": "order processed",
  "order_id": "ord_123",
  "result": "failed",
  "failure_reason": "payment_failed",
  "duration": "1234ms",
  "items_count": 3,
  "total_amount": 99.99,
  "inventory_reserved": true
}
```

You immediately see:

- The order failed at payment
- Inventory was reserved (needs cleanup)
- It took 1.2 seconds
- The order had 3 items totaling $99.99

### 3. Easy Analytics

Wide events make analytics queries simple:

```sql
-- Average order processing time by result
SELECT result, AVG(duration_ms) as avg_duration
FROM order_events
GROUP BY result;

-- Payment failure rate by method
SELECT payment_method,
       COUNT(*) FILTER (WHERE failure_reason = 'payment_failed') * 100.0 / COUNT(*) as failure_rate
FROM order_events
GROUP BY payment_method;
```

### 4. Cost Reduction

One log line with 50 fields is cheaper than 15 log lines with 3 fields each:

- Less storage
- Faster queries
- Lower ingestion costs

## What golint-sl Checks

The `wideevents` analyzer enforces:

### 1. Banned Loggers

```go
// Banned: logrus
logrus.Info("message")  // use zap instead

// Banned: stdlib log
log.Println("message")  // use zap instead

// Banned: fmt.Print for logging
fmt.Println("debug")    // use zap.Debug instead
```

### 2. Scattered Logging

```go
func Process() {
    log.Info("starting")     // Warning: scattered log #1
    // ...
    log.Info("middle step")  // Warning: scattered log #2
    // ...
    log.Info("done")         // Warning: scattered log #3
}
// Fix: Emit one wide event at the end
```

### 3. Missing Context

```go
// Warning: missing request context
logger.Info("processed",
    zap.String("user_id", userID),
)

// Good: includes correlation IDs
logger.Info("processed",
    zap.String("user_id", userID),
    zap.String("request_id", reqID),
    zap.String("trace_id", traceID),
)
```

### 4. Logs in Loops

```go
for _, item := range items {
    logger.Info("processing item", zap.String("id", item.ID))  // Log spam!
}

// Good: aggregate and log once
processedCount := 0
for _, item := range items {
    process(item)
    processedCount++
}
logger.Info("processed items", zap.Int("count", processedCount))
```

## Implementation Pattern

### Basic Template

```go
func ProcessRequest(ctx context.Context, req *Request) (result *Result, err error) {
    start := time.Now()

    // Always log, even on panic
    defer func() {
        logger.Info("request processed",
            zap.String("request_id", req.ID),
            zap.String("trace_id", getTraceID(ctx)),
            zap.Duration("duration", time.Since(start)),
            zap.Bool("success", err == nil),
            zap.Error(err),
        )
    }()

    // Business logic...
    return doWork(ctx, req)
}
```

### With Event Struct

```go
type RequestEvent struct {
    RequestID    string
    TraceID      string
    UserID       string
    Method       string
    Path         string
    StatusCode   int
    Duration     time.Duration
    BytesRead    int64
    BytesWritten int64
    Error        error
}

func (e *RequestEvent) Log(logger *zap.Logger) {
    logger.Info("http request",
        zap.String("request_id", e.RequestID),
        zap.String("trace_id", e.TraceID),
        zap.String("user_id", e.UserID),
        zap.String("method", e.Method),
        zap.String("path", e.Path),
        zap.Int("status_code", e.StatusCode),
        zap.Duration("duration", e.Duration),
        zap.Int64("bytes_read", e.BytesRead),
        zap.Int64("bytes_written", e.BytesWritten),
        zap.Error(e.Error),
    )
}
```

## OpenTelemetry Integration with otelzap

For projects using OpenTelemetry, [otelzap](https://github.com/spechtlabs/go-otel-utils) provides context-aware logging that automatically extracts trace context:

```go
import "github.com/spechtlabs/go-otel-utils/otelzap"

func ProcessRequest(ctx context.Context, req *Request) error {
    // otelzap's *Context methods automatically extract trace_id and span_id from ctx
    // No need to manually add these fields!
    otelzap.L().InfoContext(ctx, "request processed",
        zap.String("request_id", req.ID),
        zap.String("user_id", req.UserID),
        zap.Duration("duration", duration),
    )
    return nil
}
```

### Method Chaining

otelzap supports method chaining for adding error context:

```go
// WithError adds the error as a structured field
if err != nil {
    otelzap.L().WithError(err).ErrorContext(ctx, "operation failed")
}

// Equivalent to:
otelzap.L().ErrorContext(ctx, "operation failed", zap.Error(err))
```

The `wideevents` analyzer recognizes these patterns and won't flag them for "missing structured fields" or "missing request context".

## Debug Logging

The `wideevents` analyzer allows `zap.Debug` for development:

```go
func complexAlgorithm(data []int) int {
    zap.L().Debug("algorithm input", zap.Ints("data", data))

    // Debug logs are fine - they're filtered in production
    for i, v := range data {
        zap.L().Debug("iteration", zap.Int("i", i), zap.Int("v", v))
    }

    return result
}
```

Debug logs should be:

- Disabled in production (via log level)
- Used for development troubleshooting
- Not part of your observability strategy

## Further Reading

- [Logging Sucks](https://loggingsucks.com/) - The philosophy behind wide events
- [Observability Engineering](https://www.oreilly.com/library/view/observability-engineering/9781492076438/) - Broader observability concepts
- [Reference: wideevents](/reference/analyzers/wideevents) - Analyzer documentation

## Next Steps

- [Kubernetes Patterns](/understanding/kubernetes-patterns) - Controller best practices
- [Configure Analyzers](/guides/configure-analyzers) - Enable/disable wideevents
