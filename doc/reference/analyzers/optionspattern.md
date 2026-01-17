---
title: optionspattern
permalink: /reference/analyzers/optionspattern
createTime: 2025/01/16 10:00:00
---

Enforces the functional options pattern for configurable constructors.

## Category

Testability

## What It Checks

This analyzer encourages the functional options pattern for types with many configuration options.

**Constructor Threshold:** Warns when constructors (`New*` functions) have more than 4 parameters without using functional options.

**Valid Option Function Prefixes:** Functions returning Option types should use one of these prefixes:

- `With*` (primary, e.g., `WithTimeout`, `WithLogger`)
- `Allow*` (e.g., `AllowInsecure`, `AllowRetry`)
- `Enable*` (e.g., `EnableDebug`, `EnableMetrics`)
- `Disable*` (e.g., `DisableCache`, `DisableRetry`)
- `Set*` (e.g., `SetTimeout`, `SetMaxRetries`)

**Exemptions:**

- Private functions (lowercase first letter) are not checked
- `Default*` functions are exempt (they return sets of default options, not individual options)

## Why It Matters

Long parameter lists are hard to read and modify:

```go
// What do these parameters mean?
NewServer("localhost", 8080, true, false, 30, nil, nil, "v1")
```

Functional options are self-documenting:

```go
NewServer(
    WithHost("localhost"),
    WithPort(8080),
    WithTLS(true),
    WithTimeout(30 * time.Second),
)
```

## Examples

### Bad: Many Parameters

```go
func NewServer(host string, port int, tls bool, debug bool, timeout int, logger *Logger, metrics *Metrics) *Server {
    return &Server{
        host:    host,
        port:    port,
        tls:     tls,
        debug:   debug,
        timeout: timeout,
        logger:  logger,
        metrics: metrics,
    }
}
```

### Good: Functional Options

```go
type ServerOption func(*Server)

func WithHost(host string) ServerOption {
    return func(s *Server) {
        s.host = host
    }
}

func WithPort(port int) ServerOption {
    return func(s *Server) {
        s.port = port
    }
}

func WithTLS(enabled bool) ServerOption {
    return func(s *Server) {
        s.tls = enabled
    }
}

func WithTimeout(d time.Duration) ServerOption {
    return func(s *Server) {
        s.timeout = d
    }
}

func NewServer(opts ...ServerOption) *Server {
    s := &Server{
        host:    "localhost",  // Sensible defaults
        port:    8080,
        timeout: 30 * time.Second,
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

### Good: Alternative Prefixes

```go
// Allow, Enable, Disable prefixes are also valid
func AllowInsecure() ServerOption {
    return func(s *Server) {
        s.insecure = true
    }
}

func EnableDebugLogging() ServerOption {
    return func(s *Server) {
        s.debug = true
    }
}

func DisableRetry() ServerOption {
    return func(s *Server) {
        s.maxRetries = 0
    }
}

// Default* functions provide sets of defaults - exempt from prefix rule
func DefaultProductionOptions() []ServerOption {
    return []ServerOption{
        WithTLS(true),
        DisableDebug(),
        WithTimeout(30 * time.Second),
    }
}
```

### Usage

```go
// Clear and self-documenting
server := NewServer(
    WithHost("api.example.com"),
    WithPort(443),
    WithTLS(true),
    WithTimeout(60 * time.Second),
)

// Using alternative prefixes
server := NewServer(
    AllowInsecure(),
    EnableDebugLogging(),
)

// Using default options
server := NewServer(DefaultProductionOptions()...)
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  optionspattern: true  # enabled by default
```

## When to Disable

- Simple types with few configuration options
- Internal code where readability is less critical

```yaml
analyzers:
  optionspattern: false
```

## Related Analyzers

- [functionsize](/reference/analyzers/functionsize) - Function complexity
