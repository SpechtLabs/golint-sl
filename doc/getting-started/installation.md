---
title: Installation
permalink: /getting-started/installation
createTime: 2025/01/16 10:00:00
---

golint-sl runs as a [golangci-lint](https://golangci-lint.run/) module plugin. This gives you unified configuration, `nolint` directives, and seamless integration with your existing linting setup.

## Prerequisites

- **Go** 1.21 or later
- **golangci-lint** v2.0 or later ([install guide](https://golangci-lint.run/welcome/install/))

Verify golangci-lint is installed:

```bash
golangci-lint version
```

## Step 1: Create `.custom-gcl.yml`

Add this file to your project root:

```yaml
version: v2.8.0

plugins:
  - module: 'github.com/spechtlabs/golint-sl'
    version: v0.1.0  # Use latest version
```

This tells golangci-lint to build a custom binary with golint-sl included.

## Step 2: Build the Custom Binary

```bash
golangci-lint custom
```

This creates a `./custom-gcl` binary in your project directory.

::: tip
You only need to rebuild when you change the golangci-lint version or the golint-sl version in `.custom-gcl.yml`.
:::

## Step 3: Configure `.golangci.yml`

Enable the plugin in your golangci-lint configuration:

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

## Step 4: Run

```bash
./custom-gcl run ./...
```

### Verify the Plugin

Confirm golint-sl is loaded:

```bash
./custom-gcl linters | grep golint-sl
```

## Troubleshooting

### golangci-lint Not Found

Ensure golangci-lint is installed and in your `PATH`:

```bash
# Install golangci-lint v2
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.8.0
```

### Build Fails

If `golangci-lint custom` fails:

1. Ensure your Go version is 1.21 or later: `go version`
2. Ensure `go.mod` exists and modules are downloaded: `go mod download`
3. Check that the golint-sl version in `.custom-gcl.yml` exists on the module proxy

### Plugin Not Loaded

If `./custom-gcl linters | grep golint-sl` returns nothing:

1. Ensure `.golangci.yml` has `golint-sl` in the `enable` list
2. Ensure the `settings.custom.golint-sl.type` is set to `module`
3. Rebuild the binary: `golangci-lint custom`

---

::: details Standalone Binary (Advanced)
For quick local testing or development, you can also install golint-sl as a standalone binary:

```bash
go install github.com/spechtlabs/golint-sl/cmd/golint-sl@latest
golint-sl ./...
```

The standalone binary uses its own configuration file (`.golint-sl.yaml`) and CLI flags. See the [CLI Reference](/reference/cli) for details. For production use, the golangci-lint plugin is strongly recommended.
:::

## Next Steps

- [Quick Start](/getting-started/quick) - Run your first analysis
- [GitHub Actions](/guides/github-actions) - Set up CI integration
