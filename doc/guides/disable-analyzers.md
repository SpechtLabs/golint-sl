---
title: Disable Analyzers
permalink: /guides/disable-analyzers
createTime: 2025/01/16 10:00:00
---

Sometimes you need to suppress golint-sl warnings. Here's how to do it at various levels.

## Globally (Project-Wide)

Disable analyzers for your entire project in `.golangci.yml`:

```yaml
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
            - todotracker      # We use a different TODO tracking system
            - exporteddoc      # Internal project, docs not required
            - reconciler       # Not a Kubernetes project
            - statusupdate
            - sideeffects
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

Note: This suppresses warnings only on the package line itself. For true file-wide suppression, consider using the `disabled-analyzers` setting instead.

## Excluding Files

### Generated Code

golint-sl automatically skips common generated file patterns:

- `*_gen.go`
- `*.pb.go`
- `zz_generated*.go`
- Files in `mock_*` directories

### Vendor Directory

The `vendor/` directory is automatically skipped.

### Custom Exclusions via golangci-lint

Use golangci-lint's built-in exclusion mechanisms:

```yaml
# .golangci.yml
version: "2"

issues:
  exclude-dirs:
    - testdata
    - generated

  exclude-files:
    - ".*_generated\\.go$"
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

If adopting golint-sl on an existing codebase, start with a small set of analyzers and expand:

```yaml
# .golangci.yml - Week 1: Start with safety checks
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
            # Disable everything except safety checks
            - humaneerror
            - errorwrap
            - sentinelerrors
            - wideevents
            - contextlogger
            - contextpropagation
            - reconciler
            - statusupdate
            - sideeffects
            - clockinterface
            - interfaceconsistency
            - mockverify
            - optionspattern
            - httpclient
            - goroutineleak
            - nopanic
            - nestingdepth
            - syncaccess
            - closurecomplexity
            - emptyinterface
            - returninterface
            - contextfirst
            - pkgnaming
            - functionsize
            - exporteddoc
            - todotracker
            - hardcodedcreds
            - lifecycle
            - dataflow
            # nilcheck and resourceclose are NOT listed, so they stay enabled
```

Then remove analyzers from the disabled list as you fix issues, enabling more checks over time.

## Documenting Disabled Analyzers

Always document why an analyzer is disabled using YAML comments:

```yaml
settings:
  disabled-analyzers:
    # We use logrus with a custom wide-event middleware
    # See: internal/logging/README.md
    - wideevents

    # Kubernetes analyzers not applicable to CLI project
    - reconciler
    - statusupdate
    - sideeffects

    # Migrating incrementally, will enable Q2 2025
    # Tracking: JIRA-1234
    - humaneerror
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
