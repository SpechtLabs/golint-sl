// Package clockinterface provides an analyzer that enforces the use of clock interfaces
// for time operations, improving testability.
//
// Inspired by the compute-blade-agent pattern:
//
//	type Clock interface {
//	    Now() time.Time
//	    After(d time.Duration) <-chan time.Time
//	}
//
// This allows tests to control time without actually waiting.
package clockinterface

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `enforce clock interface pattern for testable time operations

This analyzer detects direct usage of time.Now() and time.After() in
business logic and suggests using an injected Clock interface instead.

The Clock interface pattern allows tests to:
- Control time without waiting
- Verify time-dependent behavior deterministically
- Avoid flaky tests due to timing issues

Example of the recommended pattern:

	type Clock interface {
	    Now() time.Time
	    After(d time.Duration) <-chan time.Time
	}

	type RealClock struct{}
	func (RealClock) Now() time.Time { return time.Now() }

	type MockClock struct { mock.Mock }
	func (m *MockClock) Now() time.Time { return m.Called().Get(0).(time.Time) }

Functions that need time should accept a Clock parameter or have it injected.`

var Analyzer = &analysis.Analyzer{
	Name:     "clockinterface",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// ExemptPackages are packages where time.Now is acceptable
var ExemptPackages = []string{
	"main",  // Entry points are fine
	"_test", // Test files are fine
}

// ExemptFunctions are function names where time.Now is acceptable
var ExemptFunctions = []string{
	"main",
	"init",
	"New", // Constructors often set default clocks
}

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Check if package is exempt
	pkgPath := pass.Pkg.Path()
	for _, exempt := range ExemptPackages {
		if strings.HasSuffix(pkgPath, exempt) || strings.Contains(pkgPath, exempt+"/") {
			return nil, nil
		}
	}

	// Track if there's a Clock interface defined
	hasClockInterface := false
	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
		(*ast.FuncDecl)(nil),
	}

	// First pass: check for Clock interface
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if ts.Name.Name == "Clock" {
				if _, ok := ts.Type.(*ast.InterfaceType); ok {
					hasClockInterface = true
				}
			}
		}
	})

	// Second pass: find time.Now() and time.After() calls
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return
		}

		// Skip exempt functions
		if fn.Name != nil {
			for _, exempt := range ExemptFunctions {
				if strings.HasPrefix(fn.Name.Name, exempt) {
					return
				}
			}
		}

		// Skip if function already accepts a Clock parameter
		if hasClockParameter(fn) {
			return
		}

		// Check function body for time calls
		if fn.Body == nil {
			return
		}

		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			ident, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			if ident.Name == "time" {
				switch sel.Sel.Name {
				case "Now":
					suggestion := "inject a Clock interface for testability"
					if hasClockInterface {
						suggestion = "use the Clock interface defined in this package"
					}
					reporter.Reportf(call.Pos(),
						"direct time.Now() call in business logic; %s", suggestion)

				case "After":
					suggestion := "inject a Clock interface with After() method"
					if hasClockInterface {
						suggestion = "use the Clock.After() method instead"
					}
					reporter.Reportf(call.Pos(),
						"direct time.After() call; %s", suggestion)

				case "Sleep":
					reporter.Reportf(call.Pos(),
						"time.Sleep() in business logic is usually a code smell; "+
							"consider using context with timeout, ticker, or returning a requeue duration")

				case "NewTicker", "NewTimer":
					reporter.Reportf(call.Pos(),
						"direct time.%s() call; consider abstracting time operations for testability",
						sel.Sel.Name)
				}
			}

			return true
		})
	})

	return nil, nil
}

// hasClockParameter checks if a function has a Clock parameter
func hasClockParameter(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil {
		return false
	}

	for _, param := range fn.Type.Params.List {
		paramType := types.ExprString(param.Type)
		if strings.Contains(paramType, "Clock") {
			return true
		}
	}

	// Also check if receiver type has a clock field
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		// We'd need type info to check struct fields, so just check naming
		recvType := types.ExprString(fn.Recv.List[0].Type)
		_ = recvType // Could enhance to check if receiver struct has clock field
	}

	return false
}

// ClockPatternInfo contains information about clock usage in a package
type ClockPatternInfo struct {
	HasClockInterface    bool
	HasRealClock         bool
	HasMockClock         bool
	DirectTimeNowCalls   int
	DirectTimeAfterCalls int
}

// AnalyzeClockPattern returns information about clock pattern usage
func AnalyzeClockPattern(pass *analysis.Pass) *ClockPatternInfo {
	info := &ClockPatternInfo{}
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
		(*ast.CallExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.TypeSpec:
			name := node.Name.Name
			if name == "Clock" {
				if _, ok := node.Type.(*ast.InterfaceType); ok {
					info.HasClockInterface = true
				}
			}
			if name == "RealClock" {
				info.HasRealClock = true
			}
			if strings.Contains(name, "MockClock") || strings.Contains(name, "FakeClock") {
				info.HasMockClock = true
			}

		case *ast.CallExpr:
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "time" {
					switch sel.Sel.Name {
					case "Now":
						info.DirectTimeNowCalls++
					case "After":
						info.DirectTimeAfterCalls++
					}
				}
			}
		}
	})

	return info
}
