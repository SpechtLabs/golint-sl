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
	"unicode"
	"unicode/utf8"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the clockinterface analyzer's documentation.
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

// Analyzer enforces an injected clock interface for testable time operations.
var Analyzer = &analysis.Analyzer{
	Name:     "clockinterface",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// ExemptPackages are package names (not paths) where time.Now is acceptable.
// An entry starting with an underscore matches as a suffix of the name, so
// "_test" covers every external test package; any other entry has to match
// the whole name.
var ExemptPackages = []string{
	"main",  // Entry points are fine
	"_test", // Test files are fine
}

// ExemptPackagePaths are package path patterns where time.Now is acceptable
// These are typically CLI/UI code where time testability is less critical
var ExemptPackagePaths = []string{
	"/cli/",      // CLI packages
	"/cmd/",      // Command packages
	"/spinner",   // Spinner UI components
	"/pretty",    // Pretty printing
	"/format",    // Formatting
	"/ui/",       // UI packages
	"/terminal/", // Terminal utilities
}

// ExemptFunctions are function names where time.Now is acceptable. A function
// is exempt when its name is an entry, or an entry followed by a new
// camel-case word: NewService and FormatAge are exempt, Newsletter and
// initializeCache are not.
var ExemptFunctions = []string{
	"main",
	"init",
	"New",     // Constructors often set default clocks
	"Format",  // Formatting functions
	"Print",   // Print functions
	"Printf",  // Print functions
	"Println", // Print functions
	"String",  // String conversion functions
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	if isExemptPackage(pass.Pkg.Name(), pass.Pkg.Path()) {
		return nil, nil
	}

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
		(*ast.FuncDecl)(nil),
	}

	// First pass: check for Clock interface
	hasClockInterface := false
	insp.Preorder(nodeFilter, func(n ast.Node) {
		if ts, ok := n.(*ast.TypeSpec); ok && isClockInterface(ts) {
			hasClockInterface = true
		}
	})

	// Second pass: find time.Now() and time.After() calls
	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return
		}

		// Skip exempt functions and functions that already accept a Clock parameter
		if isExemptFunction(fn) || hasClockParameter(fn) {
			return
		}

		// Check function body for time calls; test files are exempt like
		// external test packages are
		if fn.Body == nil || strings.HasSuffix(pass.Fset.Position(fn.Pos()).Filename, "_test.go") {
			return
		}

		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				checkTimeCall(pass, reporter, call, hasClockInterface)
			}
			return true
		})
	})

	return nil, nil
}

// isExemptPackage reports whether time.Now is acceptable in the package named
// pkgName at pkgPath.
func isExemptPackage(pkgName, pkgPath string) bool {
	for _, exempt := range ExemptPackages {
		if pkgName == exempt || (strings.HasPrefix(exempt, "_") && strings.HasSuffix(pkgName, exempt)) {
			return true
		}
	}

	for _, pattern := range ExemptPackagePaths {
		if strings.Contains(pkgPath, pattern) {
			return true
		}
	}

	return false
}

// isClockInterface reports whether ts declares an interface named Clock.
func isClockInterface(ts *ast.TypeSpec) bool {
	if ts.Name.Name != "Clock" {
		return false
	}
	_, ok := ts.Type.(*ast.InterfaceType)
	return ok
}

// isExemptFunction reports whether fn's name marks it as exempt from the check.
func isExemptFunction(fn *ast.FuncDecl) bool {
	if fn.Name == nil {
		return false
	}
	for _, exempt := range ExemptFunctions {
		rest, ok := strings.CutPrefix(fn.Name.Name, exempt)
		if !ok {
			continue
		}
		// The prefix has to be a whole camel-case word: NewService, not Newsletter
		if r, _ := utf8.DecodeRuneInString(rest); rest == "" || unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// checkTimeCall reports call if it is a direct call into the time package.
func checkTimeCall(pass *analysis.Pass, reporter *nolint.Reporter, call *ast.CallExpr, hasClockInterface bool) {
	name := timeFuncName(pass, call)

	switch name {
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
			name)
	}
}

// timeFuncName returns the name of the package-level time function call
// invokes, however the time package was imported, or "" when call is not a
// call to one. Methods such as time.Time.After are not package functions.
func timeFuncName(pass *analysis.Pass, call *ast.CallExpr) string {
	fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "time" {
		return ""
	}
	if sig, ok := fn.Type().(*types.Signature); !ok || sig.Recv() != nil {
		return ""
	}
	return fn.Name()
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
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.TypeSpec)(nil),
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
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
			switch timeFuncName(pass, node) {
			case "Now":
				info.DirectTimeNowCalls++
			case "After":
				info.DirectTimeAfterCalls++
			}
		}
	})

	return info
}
