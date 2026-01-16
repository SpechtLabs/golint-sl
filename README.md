# golint-sl - GoLint SpechtLabs

**SpechtLabs best practices for writing good Go code.**

A comprehensive Go linter with **32 analyzers** enforcing code quality, safety, architecture, and observability patterns learned from production systems.

## Installation

### Go Install

```bash
go install github.com/spechtlabs/golint-sl/cmd/golint-sl@latest
```

### Build from Source

```bash
git clone https://github.com/SpechtLabs/golint-sl.git
cd golint-sl
make install
```

### Binary Download

Download from [GitHub Releases](https://github.com/SpechtLabs/golint-sl/releases)

### Docker

```bash
docker run --rm -v $(pwd):/app -w /app ghcr.io/spechtlabs/golint-sl:latest ./...
```

## Usage

```bash
# Run all analyzers
golint-sl ./...

# Run specific analyzers
golint-sl -wideevents -contextpropagation -nilcheck ./...

# List all analyzers
golint-sl -help
```

## Analyzers (32)

### Error Handling

| Analyzer         | Description                                       |
| ---------------- | ------------------------------------------------- |
| `humaneerror`    | Enforce humane-errors-go with actionable advice   |
| `errorwrap`      | Detect bare error returns without context         |
| `sentinelerrors` | Prefer sentinel errors over inline `errors.New()` |

### Observability

| Analyzer             | Description                                      |
| -------------------- | ------------------------------------------------ |
| `wideevents`         | Enforce wide events pattern over scattered logs  |
| `contextlogger`      | Enforce context-based logging                    |
| `contextpropagation` | Ensure context is propagated through call chains |

### Kubernetes

| Analyzer       | Description                                    |
| -------------- | ---------------------------------------------- |
| `reconciler`   | Kubernetes reconciler best practices           |
| `statusupdate` | Ensure reconcilers update Status after changes |
| `sideeffects`  | SSA-based side effect detection in reconcilers |

### Testability

| Analyzer               | Description                                   |
| ---------------------- | --------------------------------------------- |
| `clockinterface`       | Abstract time operations with Clock interface |
| `interfaceconsistency` | Interface-driven design patterns              |
| `mockverify`           | Compile-time mock interface verification      |
| `optionspattern`       | Functional options pattern enforcement        |

### Resources

| Analyzer        | Description                                        |
| --------------- | -------------------------------------------------- |
| `resourceclose` | Detect unclosed resources (response bodies, files) |
| `httpclient`    | HTTP client best practices (timeouts, context)     |

### Safety

| Analyzer        | Description                               |
| --------------- | ----------------------------------------- |
| `goroutineleak` | Detect goroutines that may leak           |
| `nilcheck`      | Enforce nil checks on pointer parameters  |
| `nopanic`       | Library code must not panic               |
| `nestingdepth`  | Enforce shallow nesting and early returns |
| `syncaccess`    | Detect potential data races               |

### Clean Code

| Analyzer            | Description                                 |
| ------------------- | ------------------------------------------- |
| `varscope`          | Variables declared close to usage           |
| `closurecomplexity` | Keep closures simple, extract complex logic |
| `emptyinterface`    | Flag problematic `interface{}`/`any` usage  |
| `returninterface`   | "Accept interfaces, return structs"         |

### Architecture

| Analyzer         | Description                              |
| ---------------- | ---------------------------------------- |
| `contextfirst`   | Context should be first parameter        |
| `pkgnaming`      | Package naming conventions (no stutter)  |
| `functionsize`   | Function length limits with advice       |
| `exporteddoc`    | Exported symbols need documentation      |
| `todotracker`    | TODOs need owners                        |
| `hardcodedcreds` | Detect potential hardcoded secrets       |
| `lifecycle`      | Component lifecycle (Run/Close) patterns |
| `dataflow`       | SSA-based data flow analysis             |

## CI/CD Integration

### GitHub Actions

```yaml
- name: Run golint-sl
  run: |
    go install github.com/spechtlabs/golint-sl/cmd/golint-sl@latest
    golint-sl ./...
```

### Pre-commit

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/SpechtLabs/golint-sl
    rev: v0.1.0  # Use the latest release
    hooks:
      - id: golint-sl  # Run on all packages
      # - id: golint-sl-pkg  # Run only on changed packages (faster)
```

Available hooks:

| Hook ID | Description |
|---------|-------------|
| `golint-sl` | Run all analyzers on `./...` |
| `golint-sl-pkg` | Run only on changed Go files (faster for large repos) |

## Configuration

Create a `.golint-sl.yaml` file in your project root:

```yaml
# Configure which analyzers are enabled/disabled
analyzers:
  # Disable specific analyzers
  todotracker: false
  exporteddoc: false
  humaneerror: false
```

To disable all analyzers by default and enable only specific ones:

```yaml
analyzers:
  # Disable all by default
  default: false

  # Enable only these analyzers
  nilcheck: true
  contextfirst: true
  resourceclose: true
```

The config file is automatically discovered by searching from the current directory up to the filesystem root.

You can also use command-line flags (these override config file settings):

```bash
golint-sl -humaneerror=false ./...
golint-sl -help  # See all available flags
```

## Philosophy

**golint-sl** (GoLint SpechtLabs) enforces patterns learned from building production systems:

- [Clean Go Code](https://github.com/Pungyeon/clean-go-article) - Variable scope, early returns, function size
- [Logging Sucks](https://loggingsucks.com/) - Wide events over scattered logs
- Kubernetes best practices - Reconciler patterns, status updates
- Production experience - Context propagation, resource cleanup, nil safety

These are the coding standards we use at SpechtLabs for all Go projects.

## License

Apache 2.0

---

**GoLint SpechtLabs** - _Write Go code the right way._
