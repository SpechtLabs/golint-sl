---
title: returninterface
permalink: /reference/analyzers/returninterface
createTime: 2025/01/16 10:00:00
---

Enforces "accept interfaces, return structs" pattern.

## Category

Clean Code

## What It Checks

This analyzer detects functions that return interfaces when they should return concrete types.

## Why It Matters

The Go proverb: "Accept interfaces, return structs."

Returning interfaces:

- Limits what callers can do with the result
- Hides implementation details unnecessarily
- Makes testing harder
- Prevents callers from accessing struct fields

## Examples

### Bad: Return Interface

```go
type UserRepository interface {
    Get(id string) (*User, error)
    Save(user *User) error
}

func NewUserRepository(db *sql.DB) UserRepository {  // Returns interface
    return &userRepository{db: db}
}
```

### Good: Return Concrete Type

```go
type UserRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {  // Returns struct
    return &UserRepository{db: db}
}

func (r *UserRepository) Get(id string) (*User, error) {
    // ...
}

func (r *UserRepository) Save(user *User) error {
    // ...
}
```

### Accept Interface

```go
// Accept interface - callers can pass any implementation
func ProcessUsers(repo UserGetter, ids []string) ([]*User, error) {
    var users []*User
    for _, id := range ids {
        user, err := repo.Get(id)
        if err != nil {
            return nil, err
        }
        users = append(users, user)
    }
    return users, nil
}

type UserGetter interface {
    Get(id string) (*User, error)
}
```

### Exceptions

Factory functions for plugin systems may return interfaces:

```go
// Plugin system - interface return is appropriate
func LoadPlugin(path string) (Plugin, error) {
    // Returns interface because implementation is unknown
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  returninterface: true  # enabled by default
```

## When to Disable

- Plugin systems
- Factory functions for multiple implementations

```yaml
analyzers:
  returninterface: false
```

## Related Analyzers

- [emptyinterface](/reference/analyzers/emptyinterface) - Interface{} usage
- [interfaceconsistency](/reference/analyzers/interfaceconsistency) - Interface implementations
