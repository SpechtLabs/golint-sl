---
title: CLI Reference
permalink: /reference/cli
createTime: 2025/01/16 10:00:00
---

Complete reference for the `golint-sl` command-line interface.

## Synopsis

```bash
golint-sl [flags] [packages]
```

## Description

golint-sl analyzes Go packages for code quality, safety, and best practice violations. It uses the standard Go analysis framework and supports all common patterns.

## Packages

Specify packages using Go's package path syntax:

```bash
# Current package
golint-sl .

# Current package and all subpackages
golint-sl ./...

# Specific packages
golint-sl ./cmd/... ./internal/...

# By import path
golint-sl github.com/myorg/myproject/...
```

## Flags

### General Flags

| Flag | Description |
|------|-------------|
| `-help` | Show help message with all available flags |
| `-version` | Show version information |

### Analyzer Flags

Each analyzer can be enabled or disabled via flag:

```bash
# Enable specific analyzers (all others remain at default)
golint-sl -nilcheck -resourceclose ./...

# Disable specific analyzers
golint-sl -nilcheck=false -todotracker=false ./...

# Combine enable and disable
golint-sl -nilcheck -todotracker=false ./...
```

### Available Analyzer Flags

#### Error Handling

| Flag | Default | Description |
|------|---------|-------------|
| `-humaneerror` | enabled | Enforce humane-errors-go usage |
| `-errorwrap` | enabled | Detect bare error returns |
| `-sentinelerrors` | enabled | Prefer sentinel errors |

#### Observability

| Flag | Default | Description |
|------|---------|-------------|
| `-wideevents` | enabled | Enforce wide event logging |
| `-contextlogger` | enabled | Enforce context-based logging |
| `-contextpropagation` | enabled | Ensure context propagation |

#### Kubernetes

| Flag | Default | Description |
|------|---------|-------------|
| `-reconciler` | enabled | Kubernetes reconciler patterns |
| `-statusupdate` | enabled | Ensure status updates |
| `-sideeffects` | enabled | Detect reconciler side effects |

#### Testability

| Flag | Default | Description |
|------|---------|-------------|
| `-clockinterface` | enabled | Enforce Clock interface |
| `-interfaceconsistency` | enabled | Interface implementation checks |
| `-mockverify` | enabled | Mock interface verification |
| `-optionspattern` | enabled | Functional options pattern |

#### Resources

| Flag | Default | Description |
|------|---------|-------------|
| `-resourceclose` | enabled | Detect unclosed resources |
| `-httpclient` | enabled | HTTP client best practices |

#### Safety

| Flag | Default | Description |
|------|---------|-------------|
| `-goroutineleak` | enabled | Detect goroutine leaks |
| `-nilcheck` | enabled | Enforce nil checks |
| `-nopanic` | enabled | Library panic detection |
| `-nestingdepth` | enabled | Enforce shallow nesting |
| `-syncaccess` | enabled | Detect data races |

#### Clean Code

| Flag | Default | Description |
|------|---------|-------------|
| `-varscope` | enabled | Variable scope analysis |
| `-closurecomplexity` | enabled | Closure complexity limits |
| `-emptyinterface` | enabled | Flag interface{}/any usage |
| `-returninterface` | enabled | Return structs, not interfaces |

#### Architecture

| Flag | Default | Description |
|------|---------|-------------|
| `-contextfirst` | enabled | Context as first parameter |
| `-pkgnaming` | enabled | Package naming conventions |
| `-functionsize` | enabled | Function length limits |
| `-exporteddoc` | enabled | Exported symbol documentation |
| `-todotracker` | enabled | TODO ownership |
| `-hardcodedcreds` | enabled | Detect hardcoded secrets |
| `-lifecycle` | enabled | Component lifecycle patterns |
| `-dataflow` | enabled | SSA-based data flow analysis |

## Configuration File

golint-sl reads `.golint-sl.yaml` from the current directory or any parent directory.

See [Configuration Reference](/reference/configuration) for file format.

Command-line flags override configuration file settings.

## Output Format

golint-sl produces output compatible with Go tools:

```text
file.go:line:column: message
```

Example:

```text
./handlers/user.go:42:3: pointer parameter "user" used without nil check
./services/api.go:87:2: log call without structured fields
```

### Output to File

Redirect output to a file:

```bash
golint-sl ./... > lint-results.txt 2>&1
```

### JSON Output

golint-sl uses the standard Go analysis framework, which doesn't natively support JSON output. For JSON output, consider:

```bash
golint-sl ./... 2>&1 | your-json-converter
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No issues found |
| 1 | Issues found |
| 2 | Error (invalid flags, package errors, etc.) |

## Environment Variables

golint-sl respects standard Go environment variables:

| Variable | Description |
|----------|-------------|
| `GOPATH` | Go workspace path |
| `GOROOT` | Go installation path |
| `GO111MODULE` | Module mode (recommended: `on`) |
| `GOPROXY` | Module proxy URL |

## Examples

### Basic Usage

```bash
# Analyze all packages
golint-sl ./...

# Analyze specific package
golint-sl ./cmd/myapp

# Analyze multiple packages
golint-sl ./cmd/... ./internal/core/...
```

### Selective Analysis

```bash
# Only safety analyzers
golint-sl -nilcheck -goroutineleak -nopanic -nestingdepth -syncaccess ./...

# Only Kubernetes analyzers
golint-sl -reconciler -statusupdate -sideeffects ./...

# Disable noisy analyzers
golint-sl -todotracker=false -exporteddoc=false ./...
```

### CI Integration

```bash
# Fail on any issue
golint-sl ./... || exit 1

# Non-blocking (warning only)
golint-sl ./... || true
```

### With Make

```makefile
.PHONY: lint
lint:
 golint-sl ./...

.PHONY: lint-strict
lint-strict:
 golint-sl -todotracker -exporteddoc ./...
```

## Troubleshooting

### "package not found" Errors

Ensure modules are downloaded:

```bash
go mod download
golint-sl ./...
```

### Slow Analysis

For large codebases, disable expensive analyzers:

```bash
golint-sl -dataflow=false -sideeffects=false ./...
```

Or analyze specific packages:

```bash
golint-sl ./cmd/... ./internal/core/...
```

### Too Many Warnings

Adopt incrementally:

```yaml
# .golint-sl.yaml
analyzers:
  default: false
  nilcheck: true
  resourceclose: true
```

Then enable more analyzers as you fix issues.

## See Also

- [Configuration Reference](/reference/configuration)
- [Quick Start](/getting-started/quick)
- [GitHub Actions](/guides/github-actions)
