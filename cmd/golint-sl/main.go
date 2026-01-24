// Command golint-sl is a golangci-lint compatible linter that bundles all SpechtLabs-specific analyzers.
//
// This can be used standalone or integrated with golangci-lint as a plugin.
//
// Usage:
//
//	# Standalone
//	golint-sl ./...
//
//	# With golangci-lint (as plugin)
//	golangci-lint run --enable=golint-sl ./...
//
// Configuration:
//
// Create a .golint-sl.yaml file in your project root to configure analyzers:
//
//	analyzers:
//	  # Disable specific analyzers
//	  todotracker: false
//	  exporteddoc: false
//
//	  # Or disable all by default and enable specific ones
//	  # default: false
//	  # nilcheck: true
//	  # contextfirst: true
//
// Available analyzers (31 total):
//
// Error handling:
//   - humaneerror: Enforce humane-errors-go with actionable advice
//   - errorwrap: Detect bare error returns without context
//   - sentinelerrors: Prefer sentinel errors over inline errors.New()
//
// Observability:
//   - wideevents: Enforce wide events pattern over scattered logs
//   - contextlogger: Enforce context-based logging patterns
//   - contextpropagation: Ensure context is propagated through call chains
//
// Kubernetes:
//   - reconciler: Kubernetes reconciler best practices
//   - statusupdate: Ensure reconcilers update Status after changes
//   - sideeffects: SSA-based side effect detection in reconcilers
//
// Testability:
//   - clockinterface: Enforce Clock interface for testable time operations
//   - interfaceconsistency: Interface-driven design patterns
//   - mockverify: Ensure mocks have compile-time interface verification
//   - optionspattern: Functional options pattern enforcement
//
// Resources:
//   - resourceclose: Detect unclosed resources (response bodies, files)
//   - httpclient: Enforce http.Client best practices (timeouts)
//
// Safety:
//   - goroutineleak: Detect goroutines that may leak
//   - nilcheck: Enforce nil checks on pointer parameters
//   - nopanic: Ensure library code returns errors instead of panicking
//   - nestingdepth: Enforce shallow nesting and early returns
//   - syncaccess: Detect potential data races and synchronization issues
//
// Clean code:
//   - closurecomplexity: Detect complex anonymous functions
//   - emptyinterface: Flag problematic interface{}/any usage
//   - returninterface: Enforce "accept interfaces, return structs"
//
// Architecture:
//   - contextfirst: Ensure context.Context is first parameter
//   - pkgnaming: Enforce package naming conventions (no stutter)
//   - functionsize: Function length limits with refactoring advice
//   - exporteddoc: Ensure exported symbols have documentation
//   - todotracker: Ensure TODO/FIXME have owners
//   - hardcodedcreds: Detect potential hardcoded secrets
//   - lifecycle: Enforce component lifecycle (Run/Close) patterns
//   - dataflow: SSA-based data flow and taint analysis
package main

import (
	"fmt"
	"os"

	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/spechtlabs/golint-sl/analyzers"
	"github.com/spechtlabs/golint-sl/internal/config"
	"github.com/spechtlabs/golint-sl/internal/version"
)

func main() {
	// Handle version flag
	if len(os.Args) > 1 && (os.Args[1] == "-version" || os.Args[1] == "--version" || os.Args[1] == "version") {
		fmt.Println(version.Info())
		fmt.Println("GoLint SpechtLabs - 31 analyzers for Go best practices")
		fmt.Println("https://github.com/SpechtLabs/golint-sl")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "golint-sl: error loading config: %v\n", err)
		os.Exit(1)
	}

	// Filter analyzers based on configuration
	enabledAnalyzers := cfg.FilterAnalyzers(analyzers.All())

	if len(enabledAnalyzers) == 0 {
		fmt.Fprintf(os.Stderr, "golint-sl: no analyzers enabled (check your .golint-sl.yaml configuration)\n")
		os.Exit(1)
	}

	multichecker.Main(enabledAnalyzers...)
}
