---
title: Configuration Reference
permalink: /reference/configuration
createTime: 2025/01/16 10:00:00
---

Complete reference for golint-sl configuration.

## Configuration Files

golint-sl uses two configuration files when running as a golangci-lint module plugin:

| File | Purpose |
|------|---------|
| `.custom-gcl.yml` | Defines plugin version and golangci-lint version for the custom binary |
| `.golangci.yml` | Enables the plugin and configures analyzer settings |

## `.custom-gcl.yml`

This file tells golangci-lint how to build the custom binary with golint-sl included.

```yaml
version: v2.8.0  # golangci-lint version

plugins:
  - module: 'github.com/spechtlabs/golint-sl'
    version: v0.1.0  # golint-sl version
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `version` | Yes | golangci-lint version to use as the base binary |
| `plugins[].module` | Yes | Go module path for golint-sl |
| `plugins[].version` | Yes | Version tag of golint-sl to use |

Build the custom binary with:

```bash
golangci-lint custom
```

This creates `./custom-gcl` in your project directory.

## `.golangci.yml`

This is golangci-lint's standard configuration file. golint-sl is configured under the `linters.settings.custom` section.

### Minimal Configuration

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

### Full Configuration

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
            - todotracker
            - reconciler
            - statusupdate
            - sideeffects
```

### Plugin Settings Fields

| Field | Required | Description |
|-------|----------|-------------|
| `type` | Yes | Must be `module` |
| `description` | No | Human-readable description |
| `original-url` | No | Source URL for the plugin |
| `settings.disabled-analyzers` | No | List of analyzer names to disable |

## Plugin Settings

### disabled-analyzers

A list of analyzer names to disable. All 32 analyzers are enabled by default.

```yaml
settings:
  disabled-analyzers:
    - analyzername
```

See [Plugin Settings Reference](/reference/cli) for the complete list of analyzer names.

## Analyzer Names

All 32 analyzers and their names:

### Error Handling

| Name | Description |
|------|-------------|
| `humaneerror` | Enforce humane-errors-go |
| `errorwrap` | Detect bare error returns |
| `sentinelerrors` | Prefer sentinel errors |

### Observability

| Name | Description |
|------|-------------|
| `wideevents` | Wide event logging pattern |
| `contextlogger` | Context-based logging |
| `contextpropagation` | Context propagation |

### Kubernetes

| Name | Description |
|------|-------------|
| `reconciler` | Reconciler best practices |
| `statusupdate` | Status update requirements |
| `sideeffects` | Side effect detection |

### Testability

| Name | Description |
|------|-------------|
| `clockinterface` | Clock interface for time |
| `interfaceconsistency` | Interface implementations |
| `mockverify` | Mock interface verification |
| `optionspattern` | Functional options |

### Resources

| Name | Description |
|------|-------------|
| `resourceclose` | Resource closing |
| `httpclient` | HTTP client practices |

### Safety

| Name | Description |
|------|-------------|
| `goroutineleak` | Goroutine leak detection |
| `nilcheck` | Nil pointer checks |
| `nopanic` | Library panic prevention |
| `nestingdepth` | Nesting depth limits |
| `syncaccess` | Data race detection |

### Clean Code

| Name | Description |
|------|-------------|
| `closurecomplexity` | Closure complexity |
| `emptyinterface` | Empty interface usage |
| `returninterface` | Return type patterns |

### Architecture

| Name | Description |
|------|-------------|
| `contextfirst` | Context parameter order |
| `pkgnaming` | Package naming |
| `functionsize` | Function size limits |
| `exporteddoc` | Export documentation |
| `todotracker` | TODO tracking |
| `hardcodedcreds` | Credential detection |
| `lifecycle` | Lifecycle patterns |
| `dataflow` | Data flow analysis |

## Example Configurations

### Minimal (Accept All Defaults)

```yaml
# .golangci.yml
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

### Backend API Service

```yaml
# .golangci.yml
version: "2"

linters:
  enable:
    - golint-sl

  settings:
    custom:
      golint-sl:
        type: module
        settings:
          disabled-analyzers:
            - reconciler
            - statusupdate
            - sideeffects
```

### Kubernetes Operator

```yaml
# .golangci.yml
version: "2"

linters:
  enable:
    - golint-sl

  settings:
    custom:
      golint-sl:
        type: module
        settings:
          disabled-analyzers:
            - wideevents
            - contextlogger
```

### CLI Application

```yaml
# .golangci.yml
version: "2"

linters:
  enable:
    - golint-sl

  settings:
    custom:
      golint-sl:
        type: module
        settings:
          disabled-analyzers:
            - wideevents
            - contextlogger
            - contextpropagation
            - reconciler
            - statusupdate
            - sideeffects
```

### Library Package

```yaml
# .golangci.yml
version: "2"

linters:
  enable:
    - golint-sl

  settings:
    custom:
      golint-sl:
        type: module
        settings:
          disabled-analyzers:
            - wideevents
            - contextlogger
            - reconciler
            - statusupdate
            - sideeffects
```

## golangci-lint Issue Exclusion

Use golangci-lint's built-in exclusion mechanisms alongside golint-sl settings:

```yaml
# .golangci.yml
version: "2"

issues:
  # Exclude specific directories
  exclude-dirs:
    - testdata
    - generated

  # Exclude file patterns
  exclude-files:
    - ".*_generated\\.go$"

  # Exclude specific rules by text
  exclude-rules:
    - text: "pointer parameter .* used without nil check"
      path: "internal/legacy/"
```

## Avoiding Linter Overlap

Disable golangci-lint linters that overlap with golint-sl:

```yaml
linters:
  enable:
    - golint-sl

  disable:
    - nilerr       # Covered by golint-sl's nilcheck
    - bodyclose    # Covered by golint-sl's resourceclose
    - contextcheck # Covered by golint-sl's contextpropagation
```

---

::: details Standalone Configuration (.golint-sl.yaml)
When running as a standalone binary, golint-sl reads `.golint-sl.yaml` from the current directory or any parent directory:

```yaml
# .golint-sl.yaml
analyzers:
  default: false   # Disable all by default
  nilcheck: true   # Enable specific analyzers
  resourceclose: true
```

Command-line flags override config file settings:

```bash
golint-sl -nilcheck=false ./...
```

The standalone config is not used when running as a golangci-lint plugin.
:::

## See Also

- [Plugin Settings Reference](/reference/cli)
- [Configure Analyzers](/guides/configure-analyzers)
- [Disable Analyzers](/guides/disable-analyzers)
