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

1. Banned loggers (logrus, stdlib log, fmt.Print)
2. Single event per function (no scattered logs)
3. Structured fields on log calls
4. Request context in wide events

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

## Allowed Patterns

### Debug Logging

`zap.Debug` is allowed for development:

```go
zap.L().Debug("intermediate state", zap.Any("data", data))  // OK
```

### Test Functions

Logging in tests is not checked:

```go
func TestProcess(t *testing.T) {
    log.Println("test output")  // OK in tests
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
