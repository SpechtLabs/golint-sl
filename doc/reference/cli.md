---
title: Plugin Settings Reference
permalink: /reference/cli
createTime: 2025/01/16 10:00:00
---

Complete reference for the golint-sl plugin settings and available analyzers.

## Plugin Configuration

golint-sl is configured through `.golangci.yml` as a golangci-lint module plugin:

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
          disabled-analyzers: []
```

## Plugin Settings

### disabled-analyzers

A list of analyzer names to disable. All analyzers are enabled by default.

```yaml
settings:
  custom:
    golint-sl:
      type: module
      settings:
        disabled-analyzers:
          - todotracker
          - reconciler
```

## Running

```bash
# Run all enabled linters including golint-sl
./custom-gcl run ./...

# Run on specific packages
./custom-gcl run ./cmd/... ./internal/...

# Verbose output (shows which linters ran)
./custom-gcl run -v ./...

# List all linters including golint-sl
./custom-gcl linters
```

## Available Analyzers

All 32 analyzers and their names for use in `disabled-analyzers` and `//nolint` directives.

### Error Handling

| Name | Default | Description |
|------|---------|-------------|
| `humaneerror` | enabled | Enforce humane-errors-go usage |
| `errorwrap` | enabled | Detect bare error returns |
| `sentinelerrors` | enabled | Prefer sentinel errors |

### Observability

| Name | Default | Description |
|------|---------|-------------|
| `wideevents` | enabled | Enforce wide event logging |
| `contextlogger` | enabled | Enforce context-based logging |
| `contextpropagation` | enabled | Ensure context propagation |

### Kubernetes

| Name | Default | Description |
|------|---------|-------------|
| `reconciler` | enabled | Kubernetes reconciler patterns |
| `statusupdate` | enabled | Ensure status updates |
| `sideeffects` | enabled | Detect reconciler side effects |

### Testability

| Name | Default | Description |
|------|---------|-------------|
| `clockinterface` | enabled | Enforce Clock interface |
| `interfaceconsistency` | enabled | Interface implementation checks |
| `mockverify` | enabled | Mock interface verification |
| `optionspattern` | enabled | Functional options pattern |

### Resources

| Name | Default | Description |
|------|---------|-------------|
| `resourceclose` | enabled | Detect unclosed resources |
| `httpclient` | enabled | HTTP client best practices |

### Safety

| Name | Default | Description |
|------|---------|-------------|
| `goroutineleak` | enabled | Detect goroutine leaks |
| `nilcheck` | enabled | Enforce nil checks |
| `nopanic` | enabled | Library panic detection |
| `nestingdepth` | enabled | Enforce shallow nesting |
| `syncaccess` | enabled | Detect data races |

### Clean Code

| Name | Default | Description |
|------|---------|-------------|
| `closurecomplexity` | enabled | Closure complexity limits |
| `emptyinterface` | enabled | Flag interface{}/any usage |
| `returninterface` | enabled | Return structs, not interfaces |

### Architecture

| Name | Default | Description |
|------|---------|-------------|
| `contextfirst` | enabled | Context as first parameter |
| `pkgnaming` | enabled | Package naming conventions |
| `functionsize` | enabled | Function length limits |
| `exporteddoc` | enabled | Exported symbol documentation |
| `todotracker` | enabled | TODO ownership |
| `hardcodedcreds` | enabled | Detect hardcoded secrets |
| `lifecycle` | enabled | Component lifecycle patterns |
| `dataflow` | enabled | SSA-based data flow analysis |

## nolint Directives

When using golint-sl as a golangci-lint plugin, suppress warnings with standard `//nolint` directives using the analyzer names from the tables above:

```go
// Suppress all golint-sl analyzers
result := legacyFunc() //nolint:golint-sl

// Suppress specific analyzer
err := doSomething() //nolint:errorwrap

// Suppress multiple analyzers
data := process(x) //nolint:nilcheck,errorwrap
```

## Output Format

golint-sl produces output through golangci-lint's standard format:

```text
file.go:line:column: message (golint-sl)
```

Example:

```text
./handlers/user.go:42:3: pointer parameter "user" used without nil check (golint-sl)
./services/api.go:87:2: log call without structured fields (golint-sl)
```

## Exit Codes

Exit codes follow golangci-lint's behavior:

| Code | Meaning |
|------|---------|
| 0 | No issues found |
| 1 | Issues found |
| 2 | Warning (build errors, etc.) |
| 3 | Failure (config error, etc.) |

---

::: details Standalone CLI Reference
When running as a standalone binary (`golint-sl`), each analyzer can be toggled via CLI flags:

```bash
# Enable specific analyzers
golint-sl -nilcheck -resourceclose ./...

# Disable specific analyzers
golint-sl -nilcheck=false -todotracker=false ./...
```

The standalone binary reads `.golint-sl.yaml` for configuration. See [Configuration Reference](/reference/configuration) for the standalone config format.
:::

## See Also

- [Configuration Reference](/reference/configuration)
- [Quick Start](/getting-started/quick)
- [GitHub Actions](/guides/github-actions)
