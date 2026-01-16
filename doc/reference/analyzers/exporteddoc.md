---
title: exporteddoc
permalink: /reference/analyzers/exporteddoc
createTime: 2025/01/16 10:00:00
---

Ensures exported symbols have documentation.

## Category

Architecture

## What It Checks

This analyzer detects exported types, functions, methods, and variables without documentation comments.

## Why It Matters

Documentation is essential for:

- API usability
- IDE support (hover documentation)
- Generated documentation (godoc)
- Maintainability

## Examples

### Bad: No Documentation

```go
package user

type Service struct {  // Missing doc
    db *sql.DB
}

func New(db *sql.DB) *Service {  // Missing doc
    return &Service{db: db}
}

func (s *Service) Get(id string) (*User, error) {  // Missing doc
    // ...
}
```

### Good: With Documentation

```go
package user

// Service provides user management operations.
type Service struct {
    db *sql.DB
}

// New creates a new user Service with the given database connection.
func New(db *sql.DB) *Service {
    return &Service{db: db}
}

// Get retrieves a user by their unique identifier.
// Returns ErrNotFound if the user doesn't exist.
func (s *Service) Get(id string) (*User, error) {
    // ...
}
```

### Documentation Format

```go
// FunctionName does something specific.
//
// It handles edge cases like X and Y.
// Returns an error if Z fails.
func FunctionName(param Type) (Result, error)

// TypeName represents a thing.
//
// Use NewTypeName to create instances.
type TypeName struct {
    // Field1 is the primary identifier.
    Field1 string

    // Field2 controls behavior X.
    Field2 int
}

// ConstantName is the maximum allowed value for X.
const ConstantName = 100

// ErrNotFound is returned when the requested resource doesn't exist.
var ErrNotFound = errors.New("not found")
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  exporteddoc: true  # enabled by default
```

## When to Disable

- Internal packages
- Protobuf-generated code
- Early prototyping

```yaml
analyzers:
  exporteddoc: false
```

## Related Analyzers

- [pkgnaming](/reference/analyzers/pkgnaming) - Package naming

## See Also

- [Effective Go: Commentary](https://go.dev/doc/effective_go#commentary)
- [Go Doc Comments](https://go.dev/doc/comment)
