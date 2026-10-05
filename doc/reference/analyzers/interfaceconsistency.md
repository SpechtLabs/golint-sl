---
title: interfaceconsistency
permalink: /reference/analyzers/interfaceconsistency
createTime: 2025/01/16 10:00:00
---

Ensures a package's components depend on each other through interfaces and get their dependencies injected.

## Category

Testability

## What It Checks

This analyzer reports two things:

1. **Concrete dependency fields** - a struct field whose name contains a dependency word (`Client`, `Service`, `Repository`, `Store`, `Provider`, `Handler`, `Resolver`, `Middleware`) and whose type is a pointer to a concrete type declared in the same package. Each field is reported once, however many of the words its name contains.
2. **Dependencies created in place** - a call to a `New*` function whose name contains one of those words, made from a function that is not itself a constructor.

It does not report:

- Fields with a `json` tag; those are data (DTOs, CRDs, API types), not dependencies
- Pointers to other packages' types, such as `*http.Client` or `*sql.DB`
- Constructors (`New*` functions), whatever they return
- Test files, `main` packages (the composition root) and `NewTest*` helpers

## Why It Matters

A component that holds its collaborators as concrete types, or builds them itself, can't be tested without them. Depending on interfaces and receiving the implementations from the caller lets tests substitute fakes.

Constructors are a different matter: "accept interfaces, return structs" applies, so `NewStore` returning `*store` is fine even when the package declares a `Store` interface. See [returninterface](/reference/analyzers/returninterface).

## Examples

### Bad: Concrete Dependency Field

```go
type userRepository struct{ db *sql.DB }

type UserService struct {
    Repository *userRepository // reported
}
```

### Good: Interface Field

```go
type UserRepository interface {
    Get(id string) (*User, error)
}

type UserService struct {
    Repository UserRepository
}
```

### Bad: Creating a Dependency in Place

```go
func (s *Server) routes() {
    client := NewPaymentClient(s.cfg) // reported
    s.mux.Handle("/pay", payHandler(client))
}
```

### Good: Injecting It

```go
func NewServer(cfg Config, payments PaymentClient) *Server {
    return &Server{cfg: cfg, payments: payments}
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  interfaceconsistency: true  # enabled by default
```

## When to Disable

- Projects with minimal interface usage

```yaml
analyzers:
  interfaceconsistency: false
```

## Related Analyzers

- [mockverify](/reference/analyzers/mockverify) - Mock verification
- [returninterface](/reference/analyzers/returninterface) - Return type patterns
