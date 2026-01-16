---
title: golangci-lint Integration
permalink: /guides/golangci-lint
createTime: 2025/01/16 10:00:00
---

golint-sl can be used alongside [golangci-lint](https://golangci-lint.run/) or as a plugin.

## Running Separately

The simplest approach is to run both tools independently:

```bash
# Run golangci-lint with your existing config
golangci-lint run ./...

# Run golint-sl for production-focused checks
golint-sl ./...
```

### GitHub Actions Example

```yaml
name: Lint

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest

      - name: Install golint-sl
        run: go install github.com/spechtlabs/golint-sl/cmd/golint-sl@latest

      - name: Run golint-sl
        run: golint-sl ./...
```

## Why Both?

golangci-lint and golint-sl serve complementary purposes:

| Tool | Focus |
|------|-------|
| golangci-lint | Broad coverage: style, complexity, bugs, security |
| golint-sl | Production patterns: observability, Kubernetes, safety |

Together they provide comprehensive static analysis.

## Avoiding Overlap

Some golangci-lint linters overlap with golint-sl analyzers:

| golangci-lint | golint-sl | Recommendation |
|---------------|-----------|----------------|
| `nilerr` | `nilcheck` | Use golint-sl (more comprehensive) |
| `bodyclose` | `resourceclose` | Use golint-sl (catches more) |
| `contextcheck` | `contextpropagation` | Use golint-sl (production-focused) |

If using both, disable overlapping golangci-lint linters:

```yaml
# .golangci.yaml
linters:
  disable:
    - nilerr       # Using golint-sl's nilcheck
    - bodyclose    # Using golint-sl's resourceclose
    - contextcheck # Using golint-sl's contextpropagation
```

## Plugin Mode (Advanced)

golint-sl can theoretically be used as a golangci-lint plugin, but we recommend running them separately because:

1. **Simpler configuration** - No plugin compilation needed
2. **Independent updates** - Update each tool independently
3. **Clearer output** - Know which tool found each issue
4. **No compatibility issues** - Plugins can break with golangci-lint updates

If you still want plugin mode, see the [golangci-lint custom linters documentation](https://golangci-lint.run/contributing/new-linters/).

## Recommended Configuration

### golangci-lint

```yaml
# .golangci.yaml
linters:
  enable:
    # Style
    - gofmt
    - goimports

    # Bugs
    - staticcheck
    - gosec
    - errcheck

    # Complexity
    - cyclop
    - gocognit

  disable:
    # Using golint-sl instead
    - nilerr
    - bodyclose
    - contextcheck

linters-settings:
  cyclop:
    max-complexity: 15

  gocognit:
    min-complexity: 20
```

### golint-sl

```yaml
# .golint-sl.yaml
analyzers:
  # All enabled by default
  # Disable if not applicable to your project
  reconciler: false      # Not a Kubernetes project
  statusupdate: false
  sideeffects: false
```

## Makefile Integration

```makefile
.PHONY: lint
lint: lint-golangci lint-sl

.PHONY: lint-golangci
lint-golangci:
 golangci-lint run ./...

.PHONY: lint-sl
lint-sl:
 golint-sl ./...
```

## Pre-commit with Both

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/golangci/golangci-lint
    rev: v1.55.2
    hooks:
      - id: golangci-lint

  - repo: https://github.com/SpechtLabs/golint-sl
    rev: v0.1.0
    hooks:
      - id: golint-sl-pkg  # Fast mode for pre-commit
```

## Next Steps

- [Configure Analyzers](/guides/configure-analyzers) - Customize golint-sl
- [GitHub Actions](/guides/github-actions) - CI integration
