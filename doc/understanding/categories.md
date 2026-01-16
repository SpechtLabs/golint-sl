---
title: Analyzer Categories
permalink: /understanding/categories
createTime: 2025/01/16 10:00:00
---

golint-sl's 32 analyzers are organized into 8 categories based on the problems they solve.

## Error Handling

Ensure errors are informative and actionable.

| Analyzer | Purpose |
|----------|---------|
| `humaneerror` | Enforce [humane-errors-go](https://github.com/SierrasSoftworks/humane-errors-go) for user-friendly errors |
| `errorwrap` | Detect bare error returns that lose context |
| `sentinelerrors` | Prefer sentinel errors (`var ErrNotFound = errors.New(...)`) over inline `errors.New()` |

### Why It Matters

Bad error handling is the #1 cause of debugging nightmares:

```go
// This tells you nothing
return err

// This tells you everything
return fmt.Errorf("create user %q in org %q: %w", user.Name, org.ID, err)
```

## Observability

Make systems debuggable at scale.

| Analyzer | Purpose |
|----------|---------|
| `wideevents` | Enforce wide event logging (one log per request with rich context) |
| `contextlogger` | Ensure loggers use context for correlation |
| `contextpropagation` | Ensure context flows through all function calls |

### Why It Matters

When a request fails at 3 AM, you need:

- A single log line with all context
- Trace ID to correlate across services
- Request ID to find related events

See [Wide Events Pattern](/understanding/wide-events) for details.

## Kubernetes

Build reliable controllers and operators.

| Analyzer | Purpose |
|----------|---------|
| `reconciler` | Enforce reconciler best practices |
| `statusupdate` | Ensure status is updated after changes |
| `sideeffects` | Detect side effects in reconcilers via SSA analysis |

### Why It Matters

Kubernetes controllers have subtle requirements:

- Reconcilers must be idempotent
- Status must reflect actual state
- Side effects must be trackable

See [Kubernetes Patterns](/understanding/kubernetes-patterns) for details.

## Testability

Write code that's easy to test.

| Analyzer | Purpose |
|----------|---------|
| `clockinterface` | Abstract `time.Now()` behind an interface |
| `interfaceconsistency` | Ensure interface implementations are complete |
| `mockverify` | Verify mocks implement their interfaces at compile time |
| `optionspattern` | Enforce functional options for configurable constructors |

### Why It Matters

Hard-to-test code indicates design problems:

```go
// Untestable: time is hardcoded
func IsExpired(token *Token) bool {
    return time.Now().After(token.ExpiresAt)
}

// Testable: time is injectable
func IsExpired(token *Token, now time.Time) bool {
    return now.After(token.ExpiresAt)
}

// Even better: Clock interface
type Clock interface {
    Now() time.Time
}

func (s *Service) IsExpired(token *Token) bool {
    return s.clock.Now().After(token.ExpiresAt)
}
```

## Resources

Prevent leaks and ensure proper cleanup.

| Analyzer | Purpose |
|----------|---------|
| `resourceclose` | Detect unclosed resources (response bodies, files, connections) |
| `httpclient` | Ensure HTTP clients have timeouts |

### Why It Matters

Resource leaks are silent killers:

- HTTP response bodies leak connections
- Unclosed files exhaust file descriptors
- Missing timeouts cause goroutine leaks

```go
// Leak: response body never closed
resp, _ := http.Get(url)
data, _ := io.ReadAll(resp.Body)

// Fixed
resp, _ := http.Get(url)
defer resp.Body.Close()
data, _ := io.ReadAll(resp.Body)
```

## Safety

Prevent panics and data races.

| Analyzer | Purpose |
|----------|---------|
| `goroutineleak` | Detect goroutines that may never terminate |
| `nilcheck` | Ensure pointer parameters are checked before use |
| `nopanic` | Ensure library code returns errors instead of panicking |
| `nestingdepth` | Enforce shallow nesting with early returns |
| `syncaccess` | Detect potential data races |

### Why It Matters

Safety issues cause crashes and corruption:

- Nil dereference → panic
- Data race → corruption
- Goroutine leak → memory exhaustion

```go
// Dangerous: will panic
func Process(user *User) {
    fmt.Println(user.Name)  // nil dereference!
}

// Safe: validated first
func Process(user *User) error {
    if user == nil {
        return errors.New("user is nil")
    }
    fmt.Println(user.Name)
    return nil
}
```

## Clean Code

Keep code readable and maintainable.

| Analyzer | Purpose |
|----------|---------|
| `varscope` | Variables should be declared close to usage |
| `closurecomplexity` | Closures should be simple; extract complex logic |
| `emptyinterface` | Flag problematic `interface{}`/`any` usage |
| `returninterface` | Enforce "accept interfaces, return structs" |

### Why It Matters

Clean code is debuggable code:

```go
// Hard to follow: variable declared far from use
func Process(items []Item) {
    var total int  // Where is this used?

    // ... 50 lines of code ...

    for _, item := range items {
        total += item.Value
    }
    return total
}

// Clear: variable next to usage
func Process(items []Item) {
    // ... 50 lines of code ...

    var total int
    for _, item := range items {
        total += item.Value
    }
    return total
}
```

## Architecture

Enforce structural patterns.

| Analyzer | Purpose |
|----------|---------|
| `contextfirst` | `context.Context` should be first parameter |
| `pkgnaming` | Package names shouldn't stutter (`user.User` → `user.Entity`) |
| `functionsize` | Limit function length with refactoring suggestions |
| `exporteddoc` | Exported symbols need documentation |
| `todotracker` | TODOs need owners (`// TODO(alice): fix this`) |
| `hardcodedcreds` | Detect potential hardcoded secrets |
| `lifecycle` | Enforce component lifecycle patterns (Run/Close) |
| `dataflow` | SSA-based data flow and taint analysis |

### Why It Matters

Architecture decisions compound:

```go
// Inconsistent: context in different positions
func GetUser(id string, ctx context.Context) (*User, error)
func CreateUser(ctx context.Context, user *User) error
func DeleteUser(ctx context.Context, id string) error

// Consistent: context always first
func GetUser(ctx context.Context, id string) (*User, error)
func CreateUser(ctx context.Context, user *User) error
func DeleteUser(ctx context.Context, id string) error
```

## Choosing Analyzers

Not every analyzer applies to every project:

| Project Type | Focus On | Consider Disabling |
|--------------|----------|-------------------|
| API Service | Error handling, observability, resources | Kubernetes analyzers |
| Kubernetes Operator | Kubernetes, safety | Wide events (use controller-runtime logging) |
| CLI Tool | Error handling, safety | Context propagation, observability |
| Library | Safety, clean code, documentation | Observability, Kubernetes |

See [Configure Analyzers](/guides/configure-analyzers) for setup instructions.

## Next Steps

- [Wide Events Pattern](/understanding/wide-events) - Deep dive into observability
- [Kubernetes Patterns](/understanding/kubernetes-patterns) - Controller best practices
- [Reference: All Analyzers](/reference/analyzers/humaneerror) - Detailed analyzer documentation
