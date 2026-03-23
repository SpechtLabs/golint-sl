---
title: Configure Analyzers
permalink: /guides/configure-analyzers
createTime: 2025/01/16 10:00:00
---

Customize which analyzers run and how they behave through the golangci-lint plugin settings.

## Configuration File

All golint-sl configuration lives in `.golangci.yml` under the plugin settings:

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
```

## Disabling Specific Analyzers

All analyzers are enabled by default. Disable ones that don't apply to your project by adding them to the `disabled-analyzers` list:

```yaml
settings:
  custom:
    golint-sl:
      type: module
      settings:
        disabled-analyzers:
          # Not a Kubernetes project
          - reconciler
          - statusupdate
          - sideeffects

          # Our project uses different logging patterns
          - wideevents
          - contextlogger
```

## Project-Type Configurations

### Backend API Service

```yaml
settings:
  custom:
    golint-sl:
      type: module
      settings:
        disabled-analyzers:
          # Not a Kubernetes project
          - reconciler
          - statusupdate
          - sideeffects
```

### Kubernetes Operator

```yaml
settings:
  custom:
    golint-sl:
      type: module
      settings:
        disabled-analyzers:
          # Operators use controller-runtime logging
          - wideevents
          - contextlogger
```

### CLI Tool

```yaml
settings:
  custom:
    golint-sl:
      type: module
      settings:
        disabled-analyzers:
          # CLIs often don't use context heavily
          - contextpropagation
          # CLIs use different logging
          - wideevents
          - contextlogger
          # Not a Kubernetes project
          - reconciler
          - statusupdate
          - sideeffects
```

### Library Package

```yaml
settings:
  custom:
    golint-sl:
      type: module
      settings:
        disabled-analyzers:
          # Let consumers decide logging
          - wideevents
          - contextlogger
          # Not a Kubernetes project
          - reconciler
          - statusupdate
          - sideeffects
```

## Listing All Analyzers

See all available analyzers:

```bash
./custom-gcl linters | grep golint-sl
```

For a full list of analyzer names and descriptions, see the [Plugin Settings Reference](/reference/cli).

## Verifying Configuration

Run the linter and check which analyzers are active:

```bash
./custom-gcl run -v ./... 2>&1 | head -50
```

The verbose output shows which linters are enabled.

## Combining with Other Linters

Since golint-sl runs as a golangci-lint plugin, you can enable it alongside any other golangci-lint linter:

```yaml
version: "2"

linters:
  enable:
    - golint-sl
    - govet
    - staticcheck
    - errcheck
    - gosimple
    - ineffassign

  # Disable linters that overlap with golint-sl
  disable:
    - nilerr       # Using golint-sl's nilcheck
    - bodyclose    # Using golint-sl's resourceclose
    - contextcheck # Using golint-sl's contextpropagation

  settings:
    custom:
      golint-sl:
        type: module
        description: SpechtLabs Go linter collection
        original-url: github.com/spechtlabs/golint-sl
        settings:
          disabled-analyzers:
            - reconciler
            - statusupdate
            - sideeffects
```

## Next Steps

- [Disable Analyzers](/guides/disable-analyzers) - Suppress specific warnings
- [Reference: Configuration](/reference/configuration) - Full configuration reference
- [Reference: Analyzers](/reference/analyzers/humaneerror) - Individual analyzer documentation
