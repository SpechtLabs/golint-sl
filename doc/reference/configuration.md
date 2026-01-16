---
title: Configuration Reference
permalink: /reference/configuration
createTime: 2025/01/16 10:00:00
---

Complete reference for golint-sl configuration.

## Configuration File

golint-sl uses a YAML configuration file named `.golint-sl.yaml`.

### File Location

golint-sl searches for the configuration file by:

1. Starting from the current working directory
2. Walking up to parent directories
3. Stopping at the filesystem root

The first `.golint-sl.yaml` found is used.

### Example Locations

```text
/home/user/myproject/.golint-sl.yaml    # Project root (recommended)
/home/user/.golint-sl.yaml              # User-level default
```

## File Format

```yaml
# .golint-sl.yaml
analyzers:
  # Analyzer name: enabled (true/false)
  nilcheck: true
  wideevents: true
  reconciler: false
```

## Configuration Options

### analyzers

Map of analyzer names to enabled state.

```yaml
analyzers:
  # Enable/disable specific analyzers
  nilcheck: true
  todotracker: false
```

### analyzers.default

Special key that sets the default state for all analyzers.

```yaml
analyzers:
  # Disable all by default
  default: false

  # Then enable specific ones
  nilcheck: true
  resourceclose: true
```

If `default` is not specified, all analyzers are enabled.

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
| `varscope` | Variable scope |
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
# .golint-sl.yaml
# Empty file or no file - all analyzers enabled
```

### Backend API Service

```yaml
# .golint-sl.yaml
analyzers:
  # Disable Kubernetes-specific analyzers
  reconciler: false
  statusupdate: false
  sideeffects: false
```

### Kubernetes Operator

```yaml
# .golint-sl.yaml
analyzers:
  # Disable observability analyzers (use controller-runtime logging)
  wideevents: false
  contextlogger: false
```

### CLI Application

```yaml
# .golint-sl.yaml
analyzers:
  # Disable service-oriented analyzers
  wideevents: false
  contextlogger: false
  contextpropagation: false
  reconciler: false
  statusupdate: false
  sideeffects: false
```

### Library Package

```yaml
# .golint-sl.yaml
analyzers:
  # Focus on API quality
  nopanic: true        # Libraries must not panic
  returninterface: true
  exporteddoc: true
  emptyinterface: true

  # Disable application-specific analyzers
  wideevents: false
  contextlogger: false
  reconciler: false
  statusupdate: false
  sideeffects: false
```

### Gradual Adoption

```yaml
# .golint-sl.yaml
# Start with minimal set
analyzers:
  default: false

  # Week 1: Safety
  nilcheck: true
  resourceclose: true

  # Week 2: Add as you fix issues
  # errorwrap: true
  # sentinelerrors: true
```

### Strict Mode

```yaml
# .golint-sl.yaml
# All analyzers enabled (default behavior)
# Explicit for documentation
analyzers:
  humaneerror: true
  errorwrap: true
  sentinelerrors: true
  wideevents: true
  contextlogger: true
  contextpropagation: true
  reconciler: true
  statusupdate: true
  sideeffects: true
  clockinterface: true
  interfaceconsistency: true
  mockverify: true
  optionspattern: true
  resourceclose: true
  httpclient: true
  goroutineleak: true
  nilcheck: true
  nopanic: true
  nestingdepth: true
  syncaccess: true
  varscope: true
  closurecomplexity: true
  emptyinterface: true
  returninterface: true
  contextfirst: true
  pkgnaming: true
  functionsize: true
  exporteddoc: true
  todotracker: true
  hardcodedcreds: true
  lifecycle: true
  dataflow: true
```

## Command-Line Overrides

Command-line flags always override configuration file settings:

```bash
# Config says nilcheck: false, but enable it
golint-sl -nilcheck ./...

# Config says wideevents: true, but disable it
golint-sl -wideevents=false ./...
```

## Validation

golint-sl validates the configuration file:

- Unknown analyzer names are ignored (for forward compatibility)
- Invalid YAML causes an error
- Invalid values (non-boolean) cause an error

## Multiple Configuration Files

Only one configuration file is used (the first one found walking up from the current directory).

For monorepos with different requirements per directory:

```text
mymonorepo/
├── .golint-sl.yaml           # Default config
├── services/
│   └── api/
│       └── .golint-sl.yaml   # API-specific config
├── operators/
│   └── myoperator/
│       └── .golint-sl.yaml   # Kubernetes-specific config
└── libs/
    └── common/
        └── .golint-sl.yaml   # Library-specific config
```

Run from the appropriate directory to pick up the right config:

```bash
cd services/api && golint-sl ./...
cd operators/myoperator && golint-sl ./...
```

## See Also

- [CLI Reference](/reference/cli)
- [Configure Analyzers](/guides/configure-analyzers)
- [Disable Analyzers](/guides/disable-analyzers)
