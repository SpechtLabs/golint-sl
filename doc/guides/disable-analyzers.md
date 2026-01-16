---
title: Disable Analyzers
permalink: /guides/disable-analyzers
createTime: 2025/01/16 10:00:00
---

Sometimes you need to suppress golint-sl warnings. Here's how to do it at various levels.

## Globally (Project-Wide)

Disable an analyzer for your entire project in `.golint-sl.yaml`:

```yaml
analyzers:
  todotracker: false   # We use a different TODO tracking system
  exporteddoc: false   # Internal project, docs not required
```

## Via Command Line

Disable for a single run:

```bash
golint-sl -todotracker=false -exporteddoc=false ./...
```

## Per-Line with Directives

Use `//nolint` comment directives to suppress warnings on specific lines:

```go
// Suppress all golint-sl analyzers on this line
result := legacyFunction() //nolint:golint-sl

// Suppress a specific analyzer
err := doSomething() //nolint:errorwrap

// Suppress multiple analyzers
data := process(input) //nolint:nilcheck,errorwrap
```

The directive can also be placed on the line immediately before:

```go
//nolint:golint-sl
result := legacyFunction()
```

### Supported Formats

| Format | Description |
|--------|-------------|
| `//nolint:golint-sl` | Suppress all golint-sl analyzers |
| `//nolint:analyzername` | Suppress specific analyzer (e.g., `errorwrap`) |
| `//nolint:name1,name2` | Suppress multiple analyzers |
| `// nolint:golint-sl` | Space after `//` is allowed |

## Per-File Suppression

To suppress warnings for an entire file, use a directive at the package declaration:

```go
//nolint:golint-sl // This file uses legacy patterns intentionally
package legacy
```

Note: This suppresses warnings only on the package line itself. For true file-wide suppression, consider using the config file instead.

## Excluding Files

### Generated Code

golint-sl automatically skips common generated file patterns:

- `*_gen.go`
- `*.pb.go`
- `zz_generated*.go`
- Files in `mock_*` directories

### Vendor Directory

The `vendor/` directory is automatically skipped.

### Custom Exclusions

Use your shell to exclude files:

```bash
# Exclude specific directories
golint-sl $(go list ./... | grep -v /testdata/)

# Exclude patterns
golint-sl $(go list ./... | grep -v generated)
```

## Per-Package

Run golint-sl only on specific packages:

```bash
# Only these packages
golint-sl ./cmd/... ./internal/core/...

# Everything except tests
golint-sl $(go list ./... | grep -v /test/)
```

## When to Disable

Disable analyzers thoughtfully. Valid reasons include:

| Reason | Example |
|--------|---------|
| **Not applicable** | Kubernetes analyzers for non-K8s projects |
| **Different pattern** | Using logrus when wideevents expects zap |
| **Legacy code** | Migrating incrementally |
| **False positive** | Analyzer incorrectly flags valid code |

Invalid reasons:

- "Too many warnings" - Fix them instead
- "We've always done it this way" - Consider why the analyzer exists
- "It's just a prototype" - Prototypes become production

## Gradual Adoption

If adopting golint-sl on an existing codebase, use `default: false` and enable analyzers incrementally:

```yaml
# Week 1: Start with safety checks
analyzers:
  default: false
  nilcheck: true
  resourceclose: true

# Week 2: Add error handling
analyzers:
  default: false
  nilcheck: true
  resourceclose: true
  errorwrap: true
  sentinelerrors: true

# Week 3: Add more...
```

This prevents overwhelming the team with hundreds of warnings at once.

## Documenting Disabled Analyzers

Always document why an analyzer is disabled:

```yaml
analyzers:
  # Disabled: We use logrus with a custom wide-event middleware
  # See: internal/logging/README.md
  wideevents: false

  # Disabled: Kubernetes analyzers not applicable to CLI project
  reconciler: false
  statusupdate: false
  sideeffects: false

  # Disabled: Migrating incrementally, will enable Q2 2025
  # Tracking: JIRA-1234
  humaneerror: false
```

## Reporting False Positives

If an analyzer consistently produces false positives:

1. Check if you're using it correctly
2. Search [existing issues](https://github.com/SpechtLabs/golint-sl/issues)
3. [Open a new issue](https://github.com/SpechtLabs/golint-sl/issues/new) with:
   - Minimal reproduction code
   - Expected behavior
   - Actual behavior
   - golint-sl version

## Next Steps

- [Configure Analyzers](/guides/configure-analyzers) - Enable specific analyzers
- [Understanding Philosophy](/understanding/philosophy) - Why these patterns matter
