---
title: Installation
permalink: /getting-started/installation
createTime: 2025/01/16 10:00:00
---

golint-sl can be installed through several methods. Choose the one that fits your workflow.

## golangci-lint Plugin (Recommended)

The recommended way to use golint-sl is as a [golangci-lint](https://golangci-lint.run/) module plugin. This provides unified configuration, `nolint` directives, and seamless integration with your existing linting setup.

### Quick Setup

1. **Create `.custom-gcl.yml`** in your project:

```yaml
version: v2.8.0

plugins:
  - module: 'github.com/spechtlabs/golint-sl'
    version: v0.1.0  # Use latest version
```

1. **Build custom binary**:

```bash
golangci-lint custom
```

1. **Create `.golangci.yml`** to enable the plugin:

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

1. **Run the linter**:

```bash
./custom-gcl run ./...
```

See [golangci-lint Integration](/guides/golangci-lint) for detailed configuration options.

## Go Install (Standalone)

If you have Go installed, this is the simplest method:

```bash
go install github.com/spechtlabs/golint-sl/cmd/golint-sl@latest
```

This installs the latest version to your `$GOPATH/bin` directory (or `$GOBIN` if set).

::: tip Verify Installation

```bash
golint-sl -version
```

:::

## Pre-built Binaries

Download pre-built binaries from [GitHub Releases](https://github.com/SpechtLabs/golint-sl/releases).

### macOS

```bash
# Apple Silicon (M1/M2/M3)
curl -LO https://github.com/SpechtLabs/golint-sl/releases/latest/download/golint-sl_darwin_arm64.tar.gz
tar -xzf golint-sl_darwin_arm64.tar.gz
sudo mv golint-sl /usr/local/bin/

# Intel
curl -LO https://github.com/SpechtLabs/golint-sl/releases/latest/download/golint-sl_darwin_amd64.tar.gz
tar -xzf golint-sl_darwin_amd64.tar.gz
sudo mv golint-sl /usr/local/bin/
```

### Linux

```bash
# x86_64
curl -LO https://github.com/SpechtLabs/golint-sl/releases/latest/download/golint-sl_linux_amd64.tar.gz
tar -xzf golint-sl_linux_amd64.tar.gz
sudo mv golint-sl /usr/local/bin/

# ARM64
curl -LO https://github.com/SpechtLabs/golint-sl/releases/latest/download/golint-sl_linux_arm64.tar.gz
tar -xzf golint-sl_linux_arm64.tar.gz
sudo mv golint-sl /usr/local/bin/
```

## Docker

Run golint-sl without installing anything locally:

```bash
docker run --rm -v $(pwd):/app -w /app ghcr.io/spechtlabs/golint-sl:latest ./...
```

This mounts your current directory into the container and runs analysis on all packages.

### Docker Compose

Add to your `docker-compose.yml` for consistent CI/local development:

```yaml
services:
  lint:
    image: ghcr.io/spechtlabs/golint-sl:latest
    volumes:
      - .:/app
    working_dir: /app
    command: ./...
```

Run with:

```bash
docker compose run --rm lint
```

## Build from Source

Clone and build the latest development version:

```bash
git clone https://github.com/SpechtLabs/golint-sl.git
cd golint-sl
make install
```

This builds and installs to your `$GOPATH/bin`.

### Development Build

For development with local changes:

```bash
make build
./bin/golint-sl ./...
```

## Version Pinning

For reproducible builds, pin to a specific version:

```bash
# Go install with version
go install github.com/spechtlabs/golint-sl/cmd/golint-sl@v0.1.0

# Or use a specific Docker tag
docker run --rm -v $(pwd):/app -w /app ghcr.io/spechtlabs/golint-sl:v0.1.0 ./...
```

## Requirements

- **Go**: 1.21 or later (for `go install`)
- **Docker**: 20.10 or later (for Docker usage)

## Troubleshooting

### Command Not Found

If `golint-sl` isn't found after `go install`:

1. Ensure `$GOPATH/bin` is in your `PATH`:

   ```bash
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

2. Add to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.)

### Permission Denied (Linux)

If you get permission errors when moving to `/usr/local/bin`:

```bash
# Use sudo
sudo mv golint-sl /usr/local/bin/

# Or install to user directory
mkdir -p ~/.local/bin
mv golint-sl ~/.local/bin/
export PATH=$PATH:~/.local/bin
```

### Docker Volume Mounting Issues

If analysis fails in Docker with module errors:

```bash
# Ensure go.mod exists and modules are downloaded
go mod download

# Then run Docker
docker run --rm -v $(pwd):/app -w /app ghcr.io/spechtlabs/golint-sl:latest ./...
```

## Next Steps

- [Quick Start](/getting-started/quick) - Run your first analysis
- [GitHub Actions](/guides/github-actions) - Set up CI integration
