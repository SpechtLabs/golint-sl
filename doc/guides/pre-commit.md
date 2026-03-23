---
title: Pre-commit Hooks
permalink: /guides/pre-commit
createTime: 2025/01/16 10:00:00
---

Run golint-sl automatically before every commit to catch issues early. The hooks build a custom golangci-lint binary with golint-sl and run it on your code.

## Prerequisites

Your project must have both `.custom-gcl.yml` and `.golangci.yml` configured. See [Installation](/getting-started/installation) for setup.

## Using pre-commit Framework

[pre-commit](https://pre-commit.com/) is a framework for managing git hooks. golint-sl provides official hook definitions.

### Setup

1. Install pre-commit:

   ```bash
   # macOS
   brew install pre-commit

   # pip
   pip install pre-commit
   ```

2. Create `.pre-commit-config.yaml` in your project root:

   ```yaml
   repos:
     - repo: https://github.com/SpechtLabs/golint-sl
       rev: v0.1.0  # Use the latest release
       hooks:
         - id: golint-sl
   ```

3. Install the hooks:

   ```bash
   pre-commit install
   ```

### Available Hooks

golint-sl provides two hooks:

| Hook ID | Description | Speed |
|---------|-------------|-------|
| `golint-sl` | Builds `custom-gcl` and runs on all packages (`./...`) | Thorough |
| `golint-sl-pkg` | Builds `custom-gcl` and runs only on changed packages | Fast |

For large repositories, use `golint-sl-pkg` for faster feedback:

```yaml
repos:
  - repo: https://github.com/SpechtLabs/golint-sl
    rev: v0.1.0
    hooks:
      - id: golint-sl-pkg  # Only check changed packages
```

::: tip
The hooks require `golangci-lint` to be installed and available in your `PATH`. The custom binary is built automatically during the hook run.
:::

### Running Manually

Run all hooks on all files:

```bash
pre-commit run --all-files
```

Run only golint-sl:

```bash
pre-commit run golint-sl --all-files
```

### Skipping Hooks

For a single commit (use sparingly):

```bash
git commit --no-verify -m "WIP: quick fix"
```

::: warning
Skipping hooks should be rare. If you're skipping frequently, consider fixing the underlying issues.
:::

## Using Git Hooks Directly

If you prefer not to use pre-commit, set up git hooks manually.

### Simple Pre-commit Hook

Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash

# Build custom binary if it doesn't exist
if [ ! -f ./custom-gcl ]; then
    echo "Building custom golangci-lint with golint-sl..."
    golangci-lint custom
fi

echo "Running golint-sl..."
./custom-gcl run ./...
exit_code=$?

if [ $exit_code -ne 0 ]; then
    echo "golint-sl found issues. Please fix them before committing."
    exit 1
fi

exit 0
```

Make it executable:

```bash
chmod +x .git/hooks/pre-commit
```

### Staged Files Only

Check only staged Go files for faster feedback:

```bash
#!/bin/bash

# Get staged .go files
staged_go_files=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$')

if [ -z "$staged_go_files" ]; then
    # No Go files staged, skip
    exit 0
fi

# Build custom binary if it doesn't exist
if [ ! -f ./custom-gcl ]; then
    echo "Building custom golangci-lint with golint-sl..."
    golangci-lint custom
fi

echo "Running golint-sl on staged files..."

# Get unique package directories
packages=$(echo "$staged_go_files" | xargs -I {} dirname {} | sort -u | sed 's|^|./|')

./custom-gcl run $packages
exit_code=$?

if [ $exit_code -ne 0 ]; then
    echo "golint-sl found issues. Please fix them before committing."
    exit 1
fi

exit 0
```

### Sharing Git Hooks

Git hooks aren't versioned by default. To share hooks with your team:

1. Create a `scripts/hooks/` directory:

   ```bash
   mkdir -p scripts/hooks
   ```

2. Add your hook scripts there

3. Add setup instructions to your README with the command `git config core.hooksPath scripts/hooks`

## Configuration

golint-sl picks up your `.golangci.yml` configuration automatically. Disable analyzers that are too noisy for pre-commit:

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
            - todotracker
            - exporteddoc
```

## Troubleshooting

### Hook Not Running

Ensure the hook is installed:

```bash
# pre-commit framework
pre-commit install

# Manual hooks
ls -la .git/hooks/pre-commit
```

### golangci-lint Not Found

The hook requires golangci-lint to be installed:

```bash
# Check installation
which golangci-lint

# Install if needed
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.8.0
```

### Too Slow

For large codebases:

1. Use `golint-sl-pkg` to check only changed packages
2. Disable expensive analyzers in `.golangci.yml`:

   ```yaml
   settings:
     custom:
       golint-sl:
         type: module
         settings:
           disabled-analyzers:
             - dataflow  # SSA analysis is slower
   ```

3. Run full checks in CI instead

## Next Steps

- [GitHub Actions](/guides/github-actions) - Comprehensive CI checks
- [Configure Analyzers](/guides/configure-analyzers) - Tune which analyzers run
