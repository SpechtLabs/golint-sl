---
title: Overview
permalink: /getting-started/overview
createTime: 2025/01/16 10:00:00
---

**golint-sl** (GoLint SpechtLabs) is a comprehensive Go linter with 32 analyzers that enforce code quality, safety, architecture, and observability patterns. Unlike traditional linters that focus on syntax and formatting, golint-sl focuses on **production readiness**.

## The Problem

Most Go projects use `go vet` and maybe `golangci-lint` with a handful of linters. These catch syntax errors and common mistakes, but they miss the patterns that cause production incidents:

- Nil pointer dereferences that slip through code review
- Scattered logging that makes debugging impossible
- Resource leaks from unclosed HTTP response bodies
- Kubernetes reconcilers that forget to update status
- Context not propagated through call chains

These issues don't cause compilation errors. They cause 3 AM pages.

## The Solution

golint-sl codifies the lessons learned from building production systems at SpechtLabs. Every analyzer addresses a real pattern that has caused real problems:

| Category | Analyzers | What They Catch |
|----------|-----------|-----------------|
| Error Handling | 3 | Bare returns, missing context, inline errors |
| Observability | 3 | Scattered logs, missing context, broken propagation |
| Kubernetes | 3 | Reconciler anti-patterns, missing status updates |
| Testability | 4 | Time dependencies, interface issues, mock problems |
| Resources | 2 | Unclosed bodies, missing timeouts |
| Safety | 5 | Nil panics, goroutine leaks, data races |
| Clean Code | 4 | Variable scope, closure complexity, interfaces |
| Architecture | 8 | Context placement, naming, documentation |

## How It Works

golint-sl uses the standard Go analysis framework (`go/analysis`). It performs static analysis on your code without executing it, examining:

1. **AST (Abstract Syntax Tree)** - Structure of your code
2. **Type Information** - Types of variables and expressions
3. **SSA (Static Single Assignment)** - Data flow for complex analysis

```mermaid
flowchart LR
    A[Go Source] --> B[Parser]
    B --> C[AST]
    C --> D[Type Checker]
    D --> E[Analyzers]
    E --> F[Diagnostics]
    F --> G[Reports]
```

Each analyzer runs independently, and you can enable or disable them individually through configuration.

## Design Principles

### 1. No False Positives

Every diagnostic should indicate a real issue. If an analyzer produces too many false positives, we fix the analyzer or remove it. Your time is valuable.

### 2. Actionable Messages

Every diagnostic includes:

- A clear description of the problem
- Why it matters
- How to fix it

```text
nilcheck: pointer parameter "user" used without nil check;
          add 'if user == nil { return ... }' at function start
```

### 3. Production Focus

We don't lint for style preferences. We lint for patterns that cause production incidents. Every analyzer exists because the pattern it enforces has prevented real bugs.

### 4. Zero Configuration Required

golint-sl works out of the box with sensible defaults. You can customize it, but you shouldn't need to.

## Next Steps

- [Installation](/getting-started/installation) - Get golint-sl on your system
- [Quick Start](/getting-started/quick) - Run your first analysis
- [Philosophy](/understanding/philosophy) - Understand the patterns we enforce
