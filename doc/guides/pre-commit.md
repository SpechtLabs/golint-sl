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
| `golint-sl` | Builds `custom-gcl` and runs it on every package of every Go module | Thorough |
| `golint-sl-pkg` | Builds `custom-gcl` and runs it only on the packages of the changed files | Fast |

For large repositories, use `golint-sl-pkg` for faster feedback:

```yaml
repos:
  - repo: https://github.com/SpechtLabs/golint-sl
    rev: v0.1.0
    hooks:
      - id: golint-sl-pkg  # Only check changed packages
```

::: tip
The hooks run `golangci-lint` from your `PATH`, or through [mise](https://mise.jdx.dev/) when it isn't on your `PATH` but your project pins it in mise. The custom binary is built automatically during the hook run.
:::

### Repositories with Several Go Modules

The hooks build `custom-gcl` from the `.custom-gcl.yml` at the repository root and then run it inside each Go module: the module at the root, if there is one, and every module in a subdirectory. A repository with modules only in subdirectories and no `go.mod` at the root works the same way. Modules under `testdata`, `vendor`, and directories starting with `.` or `_` are skipped, as `go` skips them.

Each module is linted from its own directory, so golangci-lint uses the nearest `.golangci.yml` at or above it. To share one configuration between modules, put it at the repository root; with golangci-lint's default `run.relative-path-mode: cfg`, its exclusion paths stay relative to that file, so they start with the module's directory.

`golint-sl-pkg` groups the changed files by module and lints each module's changed packages in one run. Files outside every module are skipped.

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

The hooks need golangci-lint, at the version in your `.custom-gcl.yml`, on your `PATH` or pinned with mise:

```bash
# Check installation
which golangci-lint

# Pin it for the project with mise
mise use golangci-lint@2.14.0
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
