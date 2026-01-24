---
title: golangci-lint Integration
permalink: /guides/golangci-lint
createTime: 2025/01/16 10:00:00
---

golint-sl integrates with [golangci-lint](https://golangci-lint.run/) as a **module plugin** (golangci-lint v2) or can be run as a standalone tool alongside it.

## Module Plugin (Recommended)

golangci-lint v2 supports module plugins, allowing you to build a custom binary with golint-sl baked in. This provides unified configuration, `nolint` directives, and issue exclusion through golangci-lint's config.

### Building a Custom Binary

1. **Create a `.custom-gcl.yml` file** in your project root:

```yaml
version: v2.8.0

plugins:
  - module: 'github.com/spechtlabs/golint-sl'
    version: v0.1.0  # Use the latest version
```

1. **Build the custom binary**:

```bash
golangci-lint custom
```

This creates a `./custom-gcl` binary with golint-sl included.

1. **Configure golangci-lint** to enable the plugin in `.golangci.yml`:

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
          # Optional: disable specific analyzers
          disabled-analyzers:
            - todotracker      # If you don't want TODO tracking
            - reconciler       # If not a Kubernetes project
            - statusupdate
            - sideeffects
```

1. **Run the linter**:

```bash
./custom-gcl run ./...
```

### Verifying the Plugin

To verify golint-sl is loaded:

```bash
./custom-gcl linters | grep golint-sl
```

### GitHub Actions with Module Plugin

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

      - name: Install golangci-lint
        run: |
          curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.8.0

      - name: Build custom golangci-lint with golint-sl
        run: golangci-lint custom

      - name: Run linter
        run: ./custom-gcl run ./...
```

::: tip Caching the Custom Binary
For faster CI runs, you can cache the custom binary:

```yaml
- name: Cache custom golangci-lint
  uses: actions/cache@v4
  id: cache-custom-gcl
  with:
    path: ./custom-gcl
    key: custom-gcl-${{ hashFiles('.custom-gcl.yml', 'go.sum') }}

- name: Build custom golangci-lint
  if: steps.cache-custom-gcl.outputs.cache-hit != 'true'
  run: golangci-lint custom
```

:::

## Running Separately

Alternatively, run both tools independently:

```bash
# Run golangci-lint with your existing config
golangci-lint run ./...

# Run golint-sl for production-focused checks
golint-sl ./...
```

### GitHub Actions Example (Separate)

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

## Using nolint Directives

When using golint-sl as a golangci-lint plugin, you can use standard `nolint` directives:

```go
//nolint:golint-sl
func ignoredFunction() {
    // All golint-sl checks are suppressed for this function
}

// Suppress specific analyzer by name
//nolint:nilcheck
func nilNotChecked(ptr *string) {
    fmt.Println(*ptr) // nilcheck won't report this
}
```

## Available Analyzers

golint-sl includes 30 analyzers across these categories:

| Category | Analyzers |
|----------|-----------|
| Error Handling | `humaneerror`, `errorwrap`, `sentinelerrors` |
| Observability | `wideevents`, `contextlogger`, `contextpropagation` |
| Kubernetes | `reconciler`, `statusupdate`, `sideeffects` |
| Testability | `clockinterface`, `interfaceconsistency`, `mockverify`, `optionspattern` |
| Resources | `resourceclose`, `httpclient` |
| Safety | `goroutineleak`, `nilcheck`, `nopanic`, `nestingdepth`, `syncaccess` |
| Clean Code | `closurecomplexity`, `emptyinterface`, `returninterface` |
| Architecture | `contextfirst`, `pkgnaming`, `functionsize`, `exporteddoc`, `todotracker`, `hardcodedcreds`, `lifecycle`, `dataflow` |

## Disabling Specific Analyzers

When using the module plugin, you can disable specific analyzers via settings:

```yaml
# .golangci.yml
linters:
  settings:
    custom:
      golint-sl:
        type: module
        settings:
          disabled-analyzers:
            - todotracker
            - reconciler
            - statusupdate
            - sideeffects
```

When running standalone, use the golint-sl config file:

```yaml
# .golint-sl.yaml
analyzers:
  reconciler: false
  statusupdate: false
  sideeffects: false
```

## Makefile Integration

```makefile
# Using module plugin
.PHONY: lint
lint: custom-gcl
 ./custom-gcl run ./...

custom-gcl:
 golangci-lint custom

# Or running separately
.PHONY: lint-separate
lint-separate: lint-golangci lint-sl

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
