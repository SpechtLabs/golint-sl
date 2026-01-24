---
title: GitHub Actions Integration
permalink: /guides/github-actions
createTime: 2025/01/16 10:00:00
---

Run golint-sl on every pull request to catch issues before they're merged.

## Using golangci-lint Plugin (Recommended)

The recommended approach is to use golint-sl as a golangci-lint module plugin:

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

## Standalone Workflow

Alternatively, run golint-sl as a standalone tool:

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
  golint-sl:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install golint-sl
        run: go install github.com/spechtlabs/golint-sl/cmd/golint-sl@latest

      - name: Run golint-sl
        run: golint-sl ./...
```

## With Caching

Speed up runs by caching the Go module and build cache:

```yaml
name: Lint

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  golint-sl:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true

      - name: Install golint-sl
        run: go install github.com/spechtlabs/golint-sl/cmd/golint-sl@latest

      - name: Run golint-sl
        run: golint-sl ./...
```

## Using Docker

For reproducible environments, use the Docker image:

```yaml
name: Lint

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  golint-sl:
    runs-on: ubuntu-latest
    container:
      image: ghcr.io/spechtlabs/golint-sl:latest
    steps:
      - uses: actions/checkout@v4

      - name: Run golint-sl
        run: golint-sl ./...
```

## Pinned Version

Pin to a specific version for reproducibility:

```yaml
- name: Install golint-sl
  run: go install github.com/spechtlabs/golint-sl/cmd/golint-sl@v0.1.0
```

Or with Docker:

```yaml
container:
  image: ghcr.io/spechtlabs/golint-sl:v0.1.0
```

## With Configuration File

If you have a `.golint-sl.yaml` in your repository, it's automatically used:

```yaml
# .golint-sl.yaml
analyzers:
  # Disable analyzers not relevant to your project
  reconciler: false
  statusupdate: false
  sideeffects: false
```

No changes needed to the workflow - golint-sl finds and uses the config automatically.

## Alongside Other Linters

Run golint-sl as part of a comprehensive lint job:

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

      - name: Run go vet
        run: go vet ./...

      - name: Install golint-sl
        run: go install github.com/spechtlabs/golint-sl/cmd/golint-sl@latest

      - name: Run golint-sl
        run: golint-sl ./...

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
```

## Fail on Issues

By default, golint-sl exits with code 1 when issues are found, which fails the GitHub Actions job. This is the desired behavior for most projects.

To make issues non-blocking (warning only):

```yaml
- name: Run golint-sl
  run: golint-sl ./... || true
  continue-on-error: true
```

::: warning Not Recommended
Making lint failures non-blocking reduces the value of the check. Consider fixing issues instead of ignoring them.
:::

## Pull Request Annotations

GitHub Actions automatically converts tool output to PR annotations when using the standard format (which golint-sl uses). Issues appear directly on the changed lines in the PR diff.

## Matrix Testing

Run golint-sl across multiple Go versions:

```yaml
jobs:
  golint-sl:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.21', '1.22']
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}
          cache: true

      - name: Install golint-sl
        run: go install github.com/spechtlabs/golint-sl/cmd/golint-sl@latest

      - name: Run golint-sl
        run: golint-sl ./...
```

## Scheduled Runs

Run periodic full scans (useful for catching issues in dependencies):

```yaml
name: Scheduled Lint

on:
  schedule:
    - cron: '0 0 * * 0'  # Weekly on Sunday

jobs:
  golint-sl:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Install golint-sl
        run: go install github.com/spechtlabs/golint-sl/cmd/golint-sl@latest

      - name: Run golint-sl
        run: golint-sl ./...
```

## Next Steps

- [Pre-commit Hooks](/guides/pre-commit) - Catch issues before committing
- [Configuration](/reference/configuration) - Customize analyzer settings
