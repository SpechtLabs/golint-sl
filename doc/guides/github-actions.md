---
title: GitHub Actions Integration
permalink: /guides/github-actions
createTime: 2025/01/16 10:00:00
---

Run golint-sl on every pull request to catch issues before they're merged. golint-sl runs as a golangci-lint module plugin, so your CI workflow builds a custom binary and runs it.

## Basic Workflow

Create `.github/workflows/lint.yaml`:

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

This requires two config files in your repository:

**`.custom-gcl.yml`**:

```yaml
version: v2.8.0

plugins:
  - module: 'github.com/spechtlabs/golint-sl'
    version: v0.1.0
```

**`.golangci.yml`**:

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

## With Binary Caching

Building the custom binary takes time. Cache it for faster CI runs:

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

      - name: Cache custom golangci-lint
        uses: actions/cache@v4
        id: cache-custom-gcl
        with:
          path: ./custom-gcl
          key: custom-gcl-${{ hashFiles('.custom-gcl.yml', 'go.sum') }}

      - name: Build custom golangci-lint with golint-sl
        if: steps.cache-custom-gcl.outputs.cache-hit != 'true'
        run: golangci-lint custom

      - name: Run linter
        run: ./custom-gcl run ./...
```

## Pinned Versions

Pin both golangci-lint and golint-sl versions for reproducible builds. golangci-lint is pinned in the install step, and golint-sl is pinned in `.custom-gcl.yml`:

```yaml
# .custom-gcl.yml
version: v2.8.0  # Pin golangci-lint version

plugins:
  - module: 'github.com/spechtlabs/golint-sl'
    version: v0.1.0  # Pin golint-sl version
```

## With Configuration File

If you have a `.golangci.yml` with analyzer settings, it's automatically used:

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
        settings:
          disabled-analyzers:
            - reconciler
            - statusupdate
            - sideeffects
```

No changes needed to the workflow — the config is picked up automatically.

## Alongside Other Linters

Run golint-sl alongside your existing golangci-lint linters. Since golint-sl is a plugin, all linters run in a single `./custom-gcl run` invocation:

```yaml
# .golangci.yml
version: "2"

linters:
  enable:
    - golint-sl
    - govet
    - staticcheck
    - errcheck

  settings:
    custom:
      golint-sl:
        type: module
        description: SpechtLabs Go linter collection
        original-url: github.com/spechtlabs/golint-sl
```

```yaml
# In your workflow
- name: Run linter
  run: ./custom-gcl run ./...  # Runs all enabled linters including golint-sl
```

## Fail on Issues

By default, `./custom-gcl run` exits with code 1 when issues are found, which fails the GitHub Actions job. This is the desired behavior for most projects.

To make issues non-blocking (warning only):

```yaml
- name: Run linter
  run: ./custom-gcl run ./... || true
  continue-on-error: true
```

::: warning Not Recommended
Making lint failures non-blocking reduces the value of the check. Consider fixing issues instead of ignoring them.
:::

## Pull Request Annotations

GitHub Actions automatically converts tool output to PR annotations when using the standard format (which golangci-lint uses). Issues appear directly on the changed lines in the PR diff.

## Next Steps

- [Pre-commit Hooks](/guides/pre-commit) - Catch issues before committing
- [Configuration](/reference/configuration) - Customize analyzer settings
