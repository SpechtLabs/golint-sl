---
title: Configure Analyzers
permalink: /guides/configure-analyzers
createTime: 2025/01/16 10:00:00
---

Customize which analyzers run and how they behave.

## Configuration File

Create `.golint-sl.yaml` in your project root:

```yaml
analyzers:
  # Analyzer-specific settings
  nilcheck: true
  wideevents: true
  resourceclose: true
```

golint-sl automatically finds this file by searching from the current directory up to the filesystem root.

## Enable Specific Analyzers Only

Disable all analyzers by default, then enable only the ones you want:

```yaml
analyzers:
  default: false

  # Enable only these
  nilcheck: true
  resourceclose: true
  contextfirst: true
```

This is useful for:

- Gradually adopting golint-sl
- Focusing on specific categories
- Performance-sensitive environments

## Disable Specific Analyzers

All analyzers are enabled by default. Disable ones that don't apply:

```yaml
analyzers:
  # Not a Kubernetes project
  reconciler: false
  statusupdate: false
  sideeffects: false

  # Our project uses different logging patterns
  wideevents: false
  contextlogger: false
```

## Command-Line Overrides

Command-line flags override config file settings:

```bash
# Config says nilcheck: false, but run it anyway
golint-sl -nilcheck ./...

# Config says wideevents: true, but skip it
golint-sl -wideevents=false ./...
```

## Project-Type Configurations

### Backend API Service

```yaml
analyzers:
  # Critical for APIs
  nilcheck: true
  resourceclose: true
  httpclient: true
  contextpropagation: true
  errorwrap: true

  # Observability
  wideevents: true
  contextlogger: true

  # Not applicable
  reconciler: false
  statusupdate: false
  sideeffects: false
```

### Kubernetes Operator

```yaml
analyzers:
  # Kubernetes-specific
  reconciler: true
  statusupdate: true
  sideeffects: true

  # General best practices
  nilcheck: true
  contextpropagation: true
  resourceclose: true

  # Usually less relevant for operators
  wideevents: false  # Operators use controller-runtime logging
```

### CLI Tool

```yaml
analyzers:
  # Important for CLIs
  nilcheck: true
  errorwrap: true
  nopanic: true

  # Less relevant
  contextpropagation: false  # CLIs often don't use context heavily
  wideevents: false          # CLIs use different logging
  reconciler: false
  statusupdate: false
  sideeffects: false
```

### Library Package

```yaml
analyzers:
  # Critical for libraries
  nopanic: true              # Libraries must not panic
  returninterface: true      # Return concrete types
  exporteddoc: true          # Document public API
  emptyinterface: true       # Avoid interface{}

  # Less relevant
  wideevents: false          # Let consumers decide logging
  contextlogger: false
  reconciler: false
  statusupdate: false
  sideeffects: false
```

## Per-Directory Configuration

golint-sl searches for config files starting from the current directory. You can have different configs for different parts of your codebase:

```text
myproject/
├── .golint-sl.yaml          # Default config
├── cmd/
│   └── .golint-sl.yaml      # CLI-specific config
├── pkg/
│   └── .golint-sl.yaml      # Library-specific config
└── internal/
    └── operator/
        └── .golint-sl.yaml  # Kubernetes-specific config
```

## Listing All Analyzers

See all available analyzers and their descriptions:

```bash
golint-sl -help
```

Output includes:

```text
Available analyzers (32 total):

Error handling:
  humaneerror     Enforce humane-errors-go with actionable advice
  errorwrap       Detect bare error returns without context
  sentinelerrors  Prefer sentinel errors over inline errors.New()

Observability:
  wideevents         Enforce wide events pattern over scattered logs
  contextlogger      Enforce context-based logging patterns
  contextpropagation Ensure context is propagated through call chains

...
```

## Verifying Configuration

Check which analyzers will run:

```bash
# Dry run to see configuration
golint-sl -help | head -50
```

## Next Steps

- [Disable Analyzers](/guides/disable-analyzers) - Suppress specific warnings
- [Reference: Configuration](/reference/configuration) - Full configuration reference
- [Reference: Analyzers](/reference/analyzers/humaneerror) - Individual analyzer documentation
