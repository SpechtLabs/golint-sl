---
title: Quick Start
permalink: /getting-started/quick
createTime: 2025/01/16 10:00:00
---

Get golint-sl running on your project in under 5 minutes.

## Step 1: Install golangci-lint

If you don't already have golangci-lint v2:

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.8.0
```

## Step 2: Add Plugin Configuration

Create `.custom-gcl.yml` in your project root:

```yaml
version: v2.8.0

plugins:
  - module: 'github.com/spechtlabs/golint-sl'
    version: v0.1.0  # Use latest version
```

## Step 3: Build the Custom Binary

```bash
golangci-lint custom
```

## Step 4: Enable golint-sl

Create or update `.golangci.yml`:

```yaml
version: "2"

linters:
  enable:
    - golint-sl

  settings:
    custom:
      golint-sl:
        type: module
        description: SpechtLabs Go linter collection
        original-url: github.com/spechtlabs/golint-sl
```

## Step 5: Run

Navigate to your Go project and run:

```bash
./custom-gcl run ./...
```

That's it. golint-sl will analyze all packages and report any issues found.

## Understanding the Output

Output looks like standard golangci-lint results:

```text
./handlers/user.go:42:3: pointer parameter "user" used without nil check; add 'if user == nil { return ... }' at function start (golint-sl)
./services/api.go:87:2: log call without structured fields; use zap.String("field", value) to add context for wide events (golint-sl)
./controllers/reconcile.go:156:1: reconciler function does not call Status().Update(); ensure status is updated after making changes (golint-sl)
```

Each line contains:

- **File and position**: `./handlers/user.go:42:3`
- **Problem**: What the analyzer detected
- **Fix**: How to resolve it
- **Linter name**: `(golint-sl)`

## Step 6: Fix Issues

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

## Disabling Specific Analyzers

Disable analyzers that don't apply to your project via `.golangci.yml`:

```yaml
version: "2"

linters:
  enable:
    - golint-sl

  settings:
    custom:
      golint-sl:
        type: module
        description: SpechtLabs Go linter collection
        original-url: github.com/spechtlabs/golint-sl
        settings:
          disabled-analyzers:
            - reconciler       # Not a Kubernetes project
            - statusupdate
            - sideeffects
```

## Suppressing Individual Warnings

Use standard `nolint` directives:

```go
//nolint:golint-sl
func ignoredFunction() {
    // All golint-sl checks are suppressed for this function
}

//nolint:nilcheck
func nilNotChecked(ptr *string) {
    fmt.Println(*ptr) // nilcheck won't report this
}
```

## Next Steps

Now that you have golint-sl running:

- [golangci-lint Integration](/guides/golangci-lint) - Detailed configuration options
- [Configure Analyzers](/guides/configure-analyzers) - Customize which analyzers run
- [GitHub Actions](/guides/github-actions) - Add to your CI pipeline
- [Pre-commit Hooks](/guides/pre-commit) - Catch issues before committing
- [Understanding Categories](/understanding/categories) - Learn what each analyzer checks
