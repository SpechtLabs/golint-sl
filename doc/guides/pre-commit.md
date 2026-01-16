---
title: Pre-commit Hooks
permalink: /guides/pre-commit
createTime: 2025/01/16 10:00:00
---

Run golint-sl automatically before every commit to catch issues early.

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
| `golint-sl` | Runs on all packages (`./...`) | Thorough |
| `golint-sl-pkg` | Runs only on changed packages | Fast |

For large repositories, use `golint-sl-pkg` for faster feedback:

```yaml
repos:
  - repo: https://github.com/SpechtLabs/golint-sl
    rev: v0.1.0
    hooks:
      - id: golint-sl-pkg  # Only check changed packages
```

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

echo "Running golint-sl..."
golint-sl ./...
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

echo "Running golint-sl on staged files..."

# Get unique package directories
packages=$(echo "$staged_go_files" | xargs -I {} dirname {} | sort -u | sed 's|^|./|')

golint-sl $packages
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

## With Husky (Node.js Projects)

For projects using Node.js tooling, [Husky](https://typicode.github.io/husky/) is popular:

1. Install Husky:

   ```bash
   npm install husky --save-dev
   npx husky init
   ```

2. Add golint-sl hook:

   ```bash
   echo "golint-sl ./..." > .husky/pre-commit
   ```

## Configuration

golint-sl automatically uses `.golint-sl.yaml` if present:

```yaml
# .golint-sl.yaml
analyzers:
  # Disable noisy analyzers for pre-commit
  todotracker: false
  exporteddoc: false
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

### golint-sl Not Found

The hook runs in a clean environment. Ensure golint-sl is in a standard location:

```bash
# Check installation
which golint-sl

# Add to PATH in hook if needed
export PATH=$PATH:$(go env GOPATH)/bin
```

### Too Slow

For large codebases:

1. Use `golint-sl-pkg` to check only changed packages
2. Disable expensive analyzers in pre-commit config:

   ```yaml
   analyzers:
     dataflow: false  # SSA analysis is slower
   ```

3. Run full checks in CI instead

## Next Steps

- [GitHub Actions](/guides/github-actions) - Comprehensive CI checks
- [Configure Analyzers](/guides/configure-analyzers) - Tune which analyzers run
