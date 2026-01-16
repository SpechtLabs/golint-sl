---
pageLayout: home
externalLinkIcon: false

config:
  - type: doc-hero
    hero:
      name: Write production-ready Go code with confidence
      text: golint-sl
      tagline: 32 analyzers enforcing code quality, safety, architecture, and observability patterns learned from production systems.
      image: /logo.png
      actions:
        - text: Get Started
          link: /getting-started/quick
          theme: brand
          icon: mdi:rocket-launch
        - text: View Analyzers
          link: /reference/analyzers/humaneerror
          theme: alt
          icon: mdi:magnify-scan

  - type: features
    title: Why golint-sl?
    description: Patterns learned from building production systems at scale.
    features:
      - title: Production-Tested Patterns
        icon: mdi:shield-check
        details: Every analyzer enforces patterns learned from real production incidents and code reviews. No theoretical "best practices" - just what works.

      - title: Comprehensive Coverage
        icon: mdi:magnify-scan
        details: 32 analyzers covering error handling, observability, Kubernetes, testability, resource management, safety, and architecture.

      - title: Zero Configuration
        icon: mdi:cog-off
        details: Works out of the box with sensible defaults. Run `golint-sl ./...` and immediately catch issues others miss.

      - title: Actionable Feedback
        icon: mdi:message-text
        details: Every diagnostic includes a clear explanation and fix suggestion. No cryptic messages - just helpful guidance.

  - type: features
    title: Analyzer Categories
    description: Organized by the problems they solve
    features:
      - title: Error Handling
        icon: mdi:alert-circle
        details: Enforce humane errors, context wrapping, and sentinel patterns for better debugging.

      - title: Observability
        icon: mdi:chart-line
        details: Wide events pattern, context-based logging, and proper context propagation for debugging at scale.

      - title: Kubernetes
        icon: mdi:kubernetes
        details: Reconciler best practices, status updates, and side-effect detection for reliable operators.

      - title: Safety
        icon: mdi:shield
        details: Nil checks, goroutine leak detection, panic prevention, and data race detection.

      - title: Clean Code
        icon: mdi:broom
        details: Variable scope, closure complexity, interface usage, and function size limits.

      - title: Architecture
        icon: mdi:sitemap
        details: Context-first parameters, package naming, documentation requirements, and lifecycle patterns.

  - type: custom
---

## Quick Start

Get up and running in under a minute:

```bash
# Install
go install github.com/spechtlabs/golint-sl/cmd/golint-sl@latest

# Run on your project
golint-sl ./...
```

::: tip Looking for more options?
See the [Installation Guide](/getting-started/installation) for Docker, pre-built binaries, and other installation methods.
:::

## What Makes golint-sl Different?

Most linters focus on syntax and formatting. **golint-sl** focuses on **production readiness**:

:::: collapse accordion expand

- **Wide Events Over Scattered Logs** - One log line per request with 50+ structured fields beats 15 scattered logs

  ```go
  // Bad: Scattered logs throughout the function
  log.Info("starting request")
  log.Info("fetched user")
  log.Info("updated database")

  // Good: Single wide event at the end
  logger.Info("request completed",
      zap.String("request_id", reqID),
      zap.String("user_id", userID),
      zap.Duration("duration", time.Since(start)),
      zap.Int("items_processed", count),
  )
  ```

- **Nil Checks at Function Boundaries** - Catch nil pointer panics before they reach production

  ```go
  // golint-sl catches this
  func ProcessUser(user *User) error {
      return user.Save() // nil check missing!
  }

  // Fixed
  func ProcessUser(user *User) error {
      if user == nil {
          return errors.New("user cannot be nil")
      }
      return user.Save()
  }
  ```

- **Kubernetes Reconciler Patterns** - Ensure your operators follow best practices

  ```go
  // golint-sl ensures you update status
  func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
      // ... reconciliation logic ...

      // Missing status update detected!
      return ctrl.Result{}, nil
  }
  ```

::::

## Integration

golint-sl integrates with your existing workflow:

| Integration                              | Description                             |
| ---------------------------------------- | --------------------------------------- |
| [GitHub Actions](/guides/github-actions) | Run on every PR with clear annotations  |
| [Pre-commit Hooks](/guides/pre-commit)   | Catch issues before they're committed   |
| [golangci-lint](/guides/golangci-lint)   | Use as a plugin alongside other linters |
