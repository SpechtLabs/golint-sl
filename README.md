# golint-sl - GoLint SpechtLabs

**SpechtLabs best practices for writing good Go code.**

A comprehensive Go linter with **31 analyzers** enforcing code quality, safety, architecture, and observability patterns learned from production systems.

## Installation

golint-sl runs as a [golangci-lint](https://golangci-lint.run/) **module plugin** (requires golangci-lint v2).

### 1. Add `.custom-gcl.yml`

```yaml
version: v2.8.0

plugins:
  - module: 'github.com/spechtlabs/golint-sl'
    version: v0.1.0  # Use latest version
```

### 2. Build Custom Binary

```bash
golangci-lint custom
```

### 3. Configure `.golangci.yml`

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

### 4. Run

```bash
./custom-gcl run ./...
```

> **Standalone binary (for quick local testing)**
>
> ```bash
> go install github.com/spechtlabs/golint-sl/cmd/golint-sl@latest
> golint-sl ./...
> ```
>
> The standalone binary uses its own config file (`.golint-sl.yaml`) and CLI flags.
> For production use, the golangci-lint plugin is recommended.

## Usage

```bash
# Run all analyzers
./custom-gcl run ./...

# Verify plugin is loaded
./custom-gcl linters | grep golint-sl
```

## Analyzers (31)

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

## Configuration

Disable specific analyzers via `.golangci.yml`:

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
          disabled-analyzers:
            - todotracker
            - exporteddoc
            - reconciler      # Not a Kubernetes project
            - statusupdate
            - sideeffects
```

## Suppressing Warnings

Use `//nolint` directives to suppress specific warnings:

```go
// Suppress all golint-sl analyzers on this line
result := legacyFunc() //nolint:golint-sl

// Suppress specific analyzer
err := doSomething() //nolint:errorwrap

// Suppress multiple analyzers
data := process(x) //nolint:nilcheck,errorwrap
```

The directive can also be on the preceding line:

```go
//nolint:contextfirst
func Handler(w http.ResponseWriter, r *http.Request, ctx context.Context) {}
```

## CI/CD Integration

### GitHub Actions

```yaml
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
| `golint-sl` | Build custom binary and run all analyzers on `./...` |
| `golint-sl-pkg` | Build custom binary and run only on changed Go files (faster for large repos) |

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
