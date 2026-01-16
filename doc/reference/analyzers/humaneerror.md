---
title: humaneerror
permalink: /reference/analyzers/humaneerror
createTime: 2025/01/16 10:00:00
---

Enforces the use of [humane-errors-go](https://github.com/SierrasSoftworks/humane-errors-go) for user-facing errors.

## Category

Error Handling

## What It Checks

This analyzer ensures error messages provide actionable information for users, not just technical details.

## Why It Matters

Technical error messages frustrate users:

```text
Error: connection refused
```

Humane errors help users:

```text
Error: Unable to connect to the database
Hint: Check that the database server is running and accessible
Details: connection refused to localhost:5432
```

## Examples

### Bad

```go
func GetUser(id string) (*User, error) {
    user, err := db.Find(id)
    if err != nil {
        return nil, err  // Raw technical error
    }
    return user, nil
}
```

### Good

```go
import "github.com/sierrasoftworks/humane-errors-go"

func GetUser(id string) (*User, error) {
    user, err := db.Find(id)
    if err != nil {
        return nil, humane.New("Unable to find the user").
            WithHint("Check that the user ID is correct").
            WithDetails(err.Error())
    }
    return user, nil
}
```

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  humaneerror: true  # enabled by default
```

## When to Disable

- Internal services where users are developers
- CLI tools with technical audiences
- Libraries (let consumers decide error presentation)

```yaml
analyzers:
  humaneerror: false
```

## Related Analyzers

- [errorwrap](/reference/analyzers/errorwrap) - Error context wrapping
- [sentinelerrors](/reference/analyzers/sentinelerrors) - Sentinel error patterns

## See Also

- [humane-errors-go](https://github.com/SierrasSoftworks/humane-errors-go)
