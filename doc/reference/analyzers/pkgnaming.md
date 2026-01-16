---
title: pkgnaming
permalink: /reference/analyzers/pkgnaming
createTime: 2025/01/16 10:00:00
---

Enforces package naming conventions to avoid stutter.

## Category

Architecture

## What It Checks

This analyzer detects package names that cause "stutter" when used with their exported symbols.

## Why It Matters

Package stutter is redundant and verbose:

```go
user.UserService      // "user" appears twice
http.HTTPClient       // "http" appears twice
config.ConfigLoader   // "config" appears twice
```

## Examples

### Bad: Stutter

```go
// In package user
package user

type UserService struct{}      // user.UserService stutters
type UserRepository struct{}   // user.UserRepository stutters
func NewUserService() *UserService {}  // user.NewUserService stutters
```

### Good: No Stutter

```go
// In package user
package user

type Service struct{}      // user.Service is clear
type Repository struct{}   // user.Repository is clear
func NewService() *Service {}  // user.NewService is clear
```

### Bad: Package Name Repetition

```go
// In package httputil
package httputil

type HTTPClient struct{}    // httputil.HTTPClient stutters on "HTTP"
```

### Good: Clear Names

```go
// In package httputil
package httputil

type Client struct{}        // httputil.Client is clear
```

### Package Naming Guidelines

1. **Short, lowercase, no underscores**

   ```text
   Good: user, http, config
   Bad: user_service, httpClient, Config
   ```

2. **Singular, not plural**

   ```text
   Good: user (for package about users)
   Bad: users
   ```

3. **Descriptive but concise**

   ```text
   Good: auth, storage, cache
   Bad: authentication, datastorage, memorycache
   ```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  pkgnaming: true  # enabled by default
```

## When to Disable

- Generated code with fixed names
- Compatibility with external conventions

```yaml
analyzers:
  pkgnaming: false
```

## Related Analyzers

- [exporteddoc](/reference/analyzers/exporteddoc) - Export documentation

## See Also

- [Effective Go: Package names](https://go.dev/doc/effective_go#package-names)
