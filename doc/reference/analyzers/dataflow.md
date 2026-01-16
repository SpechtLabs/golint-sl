---
title: dataflow
permalink: /reference/analyzers/dataflow
createTime: 2025/01/16 10:00:00
---

SSA-based data flow and taint analysis.

## Category

Architecture

## What It Checks

This analyzer uses Static Single Assignment (SSA) form to perform:

- Data flow analysis
- Taint tracking
- Value propagation analysis

## Why It Matters

Data flow analysis can detect:

- Sensitive data flowing to insecure outputs
- Unvalidated input reaching critical operations
- Data dependencies that affect correctness

## How It Works

SSA transforms code into a form where each variable is assigned exactly once, making data flow explicit:

```go
// Original code
x := 1
x = x + 1
y := x

// SSA form (conceptual)
x1 := 1
x2 := x1 + 1
y1 := x2
```

This makes it possible to track where values come from and where they flow to.

## Examples

### Detected: Tainted Data

```go
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    userInput := r.URL.Query().Get("data")  // Tainted source

    // Tainted data flows directly to output
    fmt.Fprintf(w, userInput)  // Potential XSS!
}
```

### Good: Sanitized

```go
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    userInput := r.URL.Query().Get("data")  // Tainted source

    // Sanitize before output
    safe := html.EscapeString(userInput)
    fmt.Fprintf(w, safe)  // OK
}
```

### Detected: Sensitive Data Logged

```go
func Login(username, password string) {
    log.Printf("Login attempt: %s:%s", username, password)  // Password logged!
}
```

## Performance

SSA analysis is more expensive than AST analysis. For large codebases, you may want to:

- Run it only in CI, not pre-commit
- Disable it for faster local development

## Configuration

```yaml
# .golint-sl.yaml
analyzers:
  dataflow: true  # enabled by default
```

## When to Disable

- Performance-sensitive local development
- Projects where data flow patterns are simple

```yaml
analyzers:
  dataflow: false
```

## Related Analyzers

- [sideeffects](/reference/analyzers/sideeffects) - SSA-based side effect detection
- [hardcodedcreds](/reference/analyzers/hardcodedcreds) - Credential detection

## See Also

- [golang.org/x/tools/go/ssa](https://pkg.go.dev/golang.org/x/tools/go/ssa)
