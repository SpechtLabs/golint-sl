---
title: Quick Start
permalink: /getting-started/quick
createTime: 2025/01/16 10:00:00
---

Get golint-sl running on your project in under 5 minutes.

## Step 1: Install

```bash
go install github.com/spechtlabs/golint-sl/cmd/golint-sl@latest
```

## Step 2: Run

Navigate to your Go project and run:

```bash
golint-sl ./...
```

That's it. golint-sl will analyze all packages and report any issues found.

## Understanding the Output

golint-sl produces output similar to `go vet`:

```text
./handlers/user.go:42:3: pointer parameter "user" used without nil check; add 'if user == nil { return ... }' at function start
./services/api.go:87:2: log call without structured fields; use zap.String("field", value) to add context for wide events
./controllers/reconcile.go:156:1: reconciler function does not call Status().Update(); ensure status is updated after making changes
```

Each line contains:

- **File and position**: `./handlers/user.go:42:3`
- **Problem**: What the analyzer detected
- **Fix**: How to resolve it

## Step 3: Fix Issues

Work through the reported issues. Each diagnostic tells you exactly what to fix:

### Nil Check Example

Before:

```go
func ProcessUser(user *User) error {
    return user.Save() // Panic if user is nil!
}
```

After:

```go
func ProcessUser(user *User) error {
    if user == nil {
        return errors.New("user cannot be nil")
    }
    return user.Save()
}
```

### Structured Logging Example

Before:

```go
log.Info("user created")
```

After:

```go
logger.Info("user created",
    zap.String("user_id", user.ID),
    zap.String("request_id", ctx.Value("request_id").(string)),
)
```

## Running Specific Analyzers

Run only certain analyzers:

```bash
# Only nil checks and resource closing
golint-sl -nilcheck -resourceclose ./...

# Only Kubernetes-related analyzers
golint-sl -reconciler -statusupdate -sideeffects ./...
```

## Listing Available Analyzers

See all 32 analyzers:

```bash
golint-sl -help
```

## Configuration File

Create `.golint-sl.yaml` in your project root to configure defaults:

```yaml
analyzers:
  # Disable analyzers that don't apply to your project
  reconciler: false      # Not a Kubernetes project
  statusupdate: false
  sideeffects: false

  # These are enabled by default, but you can be explicit
  nilcheck: true
  resourceclose: true
```

See [Configuration](/reference/configuration) for all options.

## Exit Codes

golint-sl uses standard exit codes:

| Code | Meaning |
|------|---------|
| 0 | No issues found |
| 1 | Issues found |
| 2 | Error (invalid flags, etc.) |

Use this in CI to fail builds on issues:

```bash
golint-sl ./... || exit 1
```

## Next Steps

Now that you have golint-sl running:

- [Configure Analyzers](/guides/configure-analyzers) - Customize which analyzers run
- [GitHub Actions](/guides/github-actions) - Add to your CI pipeline
- [Pre-commit Hooks](/guides/pre-commit) - Catch issues before committing
- [Understanding Categories](/understanding/categories) - Learn what each analyzer checks
