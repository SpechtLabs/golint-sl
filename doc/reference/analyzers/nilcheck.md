---
title: nilcheck
permalink: /reference/analyzers/nilcheck
createTime: 2025/01/16 10:00:00
---

Enforces nil checks on pointer parameters before use.

## Category

Safety

## What It Checks

This analyzer detects pointer parameters that are used without being checked for nil first.

## Why It Matters

Nil pointer dereferences cause panics:

```text
panic: runtime error: invalid memory address or nil pointer dereference
```

Explicit nil checks provide better error messages and prevent crashes.

## Examples

### Bad

```go
func ProcessUser(user *User) error {
    // Panics if user is nil!
    return user.Save()
}
```

### Good

```go
func ProcessUser(user *User) error {
    if user == nil {
        return errors.New("user cannot be nil")
    }
    return user.Save()
}
```

### Bad: Multiple Usages

```go
func UpdateProfile(user *User, profile *Profile) error {
    user.Name = profile.Name      // Both could panic!
    user.Email = profile.Email
    return user.Save()
}
```

### Good: Check All Pointers

```go
func UpdateProfile(user *User, profile *Profile) error {
    if user == nil {
        return errors.New("user cannot be nil")
    }
    if profile == nil {
        return errors.New("profile cannot be nil")
    }
    user.Name = profile.Name
    user.Email = profile.Email
    return user.Save()
}
```

## Trusted Types

The analyzer skips certain types that are guaranteed non-nil by their frameworks:

- `*testing.T`, `*testing.B`, `*testing.M`
- `*gin.Context`, `*gin.Engine`
- `*cobra.Command`
- `*http.Request`, `http.ResponseWriter`
- `context.Context`

## Skipped Files

The analyzer skips generated files:

- `*_gen.go`
- `*.pb.go`
- `zz_generated*.go`
- Files in `mock_*` directories

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  nilcheck: true  # enabled by default
```

## When to Disable

This analyzer should rarely be disabled. Nil checks are fundamental safety.

```yaml
analyzers:
  nilcheck: false  # Not recommended
```

## Related Analyzers

- [nopanic](/reference/analyzers/nopanic) - Prevent panics in libraries
- [errorwrap](/reference/analyzers/errorwrap) - Error handling
