// Package sideeffects provides an SSA-based analyzer that detects unwanted side effects
// in specific contexts, such as:
// - Kubernetes reconcilers making HTTP calls directly
// - Controllers writing to global state
// - Functions that should be pure performing I/O
package sideeffects

import (
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `detect unwanted side effects using SSA analysis

This analyzer uses SSA (Static Single Assignment) form to track data flow and detect:
1. Reconcilers making direct HTTP calls (should go through service layer)
2. Controllers accessing database directly (should use repository pattern)
3. Pure functions performing I/O operations
4. Global state mutations in handler functions

SSA provides a more accurate view of program flow than AST alone.`

var Analyzer = &analysis.Analyzer{
	Name:     "sideeffects",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{buildssa.Analyzer},
	Run:      run,
}

// Configuration for what constitutes forbidden side effects
type Config struct {
	// ForbiddenCallsInReconcilers are function patterns that shouldn't be called from reconcilers
	ForbiddenCallsInReconcilers []string
	// ForbiddenImportsInControllers are packages that controllers shouldn't import directly
	ForbiddenImportsInControllers []string
	// PureFunctionPatterns are function name patterns that should have no side effects
	PureFunctionPatterns []string
}

var defaultConfig = Config{
	ForbiddenCallsInReconcilers: []string{
		"net/http.Get",
		"net/http.Post",
		"net/http.Do",
		"database/sql.Open",
		"database/sql.(*DB).Exec",
		"database/sql.(*DB).Query",
	},
	ForbiddenImportsInControllers: []string{
		"database/sql",
		"net/http",
	},
	PureFunctionPatterns: []string{
		"*Validator",
		"*Parser",
		"*Formatter",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	ssaInfo := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)

	for _, fn := range ssaInfo.SrcFuncs {
		// Check if this is a reconciler function
		if isReconcilerFunc(fn) {
			checkReconcilerSideEffects(reporter, fn)
		}

		// Check if this function should be pure
		if shouldBePure(fn) {
			checkPureFunctionSideEffects(reporter, fn)
		}

		// Check for global state mutations in handlers
		if isHandlerFunc(fn) {
			checkHandlerGlobalMutations(reporter, fn)
		}
	}

	return nil, nil
}

// isReconcilerFunc checks if a function is a Kubernetes reconciler
func isReconcilerFunc(fn *ssa.Function) bool {
	if fn == nil || fn.Signature == nil {
		return false
	}

	// Check method receiver for Reconciler pattern
	recv := fn.Signature.Recv()
	if recv == nil {
		return false
	}

	recvType := recv.Type().String()

	// Common patterns for reconcilers
	patterns := []string{
		"Reconciler",
		"Controller",
		"*KubeOperator",
	}

	for _, pattern := range patterns {
		if strings.Contains(recvType, pattern) {
			return true
		}
	}

	// Check if the function name is "Reconcile"
	return fn.Name() == "Reconcile"
}

// checkReconcilerSideEffects analyzes a reconciler function for forbidden calls
func checkReconcilerSideEffects(reporter *nolint.Reporter, fn *ssa.Function) {
	// Walk all blocks in the function
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}

			callee := call.Call.StaticCallee()
			if callee == nil {
				continue
			}

			calleeName := callee.String()

			// Check against forbidden calls
			for _, forbidden := range defaultConfig.ForbiddenCallsInReconcilers {
				if strings.Contains(calleeName, forbidden) || matchesCallPattern(callee, forbidden) {
					reporter.Reportf(call.Pos(),
						"reconciler should not make direct %s call; use service layer abstraction",
						forbidden)
				}
			}

			// Check for HTTP client usage
			if isHTTPClientCall(callee) {
				reporter.Reportf(call.Pos(),
					"reconciler should not make HTTP calls directly; inject an HTTP client interface")
			}

			// Check for database calls
			if isDatabaseCall(callee) {
				reporter.Reportf(call.Pos(),
					"reconciler should not access database directly; use repository pattern")
			}
		}
	}
}

// shouldBePure checks if a function should be pure based on naming conventions
func shouldBePure(fn *ssa.Function) bool {
	name := fn.Name()
	for _, pattern := range defaultConfig.PureFunctionPatterns {
		pattern = strings.TrimPrefix(pattern, "*")
		if strings.Contains(name, pattern) {
			return true
		}
	}

	// Functions named "validate*", "parse*", "format*" should be pure
	lowerName := strings.ToLower(name)
	purePatterns := []string{"validate", "parse", "format", "compute", "calculate", "convert"}
	for _, p := range purePatterns {
		if strings.HasPrefix(lowerName, p) {
			return true
		}
	}

	return false
}

// checkPureFunctionSideEffects ensures pure functions don't have I/O side effects
func checkPureFunctionSideEffects(reporter *nolint.Reporter, fn *ssa.Function) {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}

			callee := call.Call.StaticCallee()
			if callee == nil {
				continue
			}

			// Check for I/O operations
			if isIOOperation(callee) {
				reporter.Reportf(call.Pos(),
					"function %q should be pure but contains I/O operation %s",
					fn.Name(), callee.Name())
			}

			// Check for time-dependent operations
			if isTimeDependentCall(callee) {
				reporter.Reportf(call.Pos(),
					"function %q should be pure but depends on time; accept time as parameter instead",
					fn.Name())
			}
		}
	}
}

// isHandlerFunc checks if this is an HTTP handler or controller method
func isHandlerFunc(fn *ssa.Function) bool {
	if fn == nil || fn.Signature == nil {
		return false
	}

	// Check for gin.Context parameter
	params := fn.Signature.Params()
	for i := 0; i < params.Len(); i++ {
		param := params.At(i)
		paramType := param.Type().String()
		if strings.Contains(paramType, "gin.Context") ||
			strings.Contains(paramType, "http.ResponseWriter") {
			return true
		}
	}

	return false
}

// checkHandlerGlobalMutations checks for global state mutations in handlers
func checkHandlerGlobalMutations(reporter *nolint.Reporter, fn *ssa.Function) {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			// Check for stores to global variables
			store, ok := instr.(*ssa.Store)
			if !ok {
				continue
			}

			// Check if the address is a global
			if global, ok := store.Addr.(*ssa.Global); ok {
				reporter.Reportf(store.Pos(),
					"handler function should not mutate global state %q; use dependency injection",
					global.Name())
			}
		}
	}
}

// matchesCallPattern checks if a callee matches a forbidden pattern
func matchesCallPattern(callee *ssa.Function, pattern string) bool {
	if callee.Pkg != nil {
		fullName := callee.Pkg.Pkg.Path() + "." + callee.Name()
		return strings.Contains(fullName, pattern)
	}
	return false
}

// isHTTPClientCall checks if a function is an HTTP client call
func isHTTPClientCall(fn *ssa.Function) bool {
	if fn.Pkg == nil {
		return false
	}

	pkgPath := fn.Pkg.Pkg.Path()
	if pkgPath == "net/http" {
		httpMethods := []string{"Get", "Post", "Head", "Put", "Delete", "Do"}
		for _, method := range httpMethods {
			if fn.Name() == method {
				return true
			}
		}
	}

	return false
}

// isDatabaseCall checks if a function is a database call
func isDatabaseCall(fn *ssa.Function) bool {
	if fn.Pkg == nil {
		return false
	}

	pkgPath := fn.Pkg.Pkg.Path()
	dbPackages := []string{"database/sql", "gorm.io", "go.mongodb.org"}
	for _, pkg := range dbPackages {
		if strings.HasPrefix(pkgPath, pkg) {
			return true
		}
	}

	return false
}

// isIOOperation checks if a function performs I/O
func isIOOperation(fn *ssa.Function) bool {
	if fn.Pkg == nil {
		return false
	}

	pkgPath := fn.Pkg.Pkg.Path()
	ioPkgs := []string{"os", "io", "net", "bufio", "database/sql"}
	for _, pkg := range ioPkgs {
		if strings.HasPrefix(pkgPath, pkg) {
			ioFuncs := []string{
				"Read", "Write", "Open", "Create", "Remove", "Mkdir",
				"Stat", "Chmod", "Chown", "Dial", "Listen", "Accept",
			}
			for _, f := range ioFuncs {
				if strings.Contains(fn.Name(), f) {
					return true
				}
			}
		}
	}

	return false
}

// isTimeDependentCall checks if a function depends on current time
func isTimeDependentCall(fn *ssa.Function) bool {
	if fn.Pkg == nil {
		return false
	}

	if fn.Pkg.Pkg.Path() == "time" {
		timeFuncs := []string{"Now", "Since", "Until"}
		for _, f := range timeFuncs {
			if fn.Name() == f {
				return true
			}
		}
	}

	return false
}

// TrackDataFlow traces the flow of a value through the SSA graph
// This is useful for understanding where sensitive data might leak
func TrackDataFlow(fn *ssa.Function, value ssa.Value) []ssa.Instruction {
	var flow []ssa.Instruction

	// Get all referrers (uses) of this value
	refs := value.Referrers()
	if refs == nil {
		return flow
	}

	for _, ref := range *refs {
		flow = append(flow, ref)

		// If the referrer produces a new value, track that too
		if instr, ok := ref.(ssa.Value); ok {
			flow = append(flow, TrackDataFlow(fn, instr)...)
		}
	}

	return flow
}

// CheckSensitiveDataLeak uses SSA to track if sensitive data might leak
func CheckSensitiveDataLeak(reporter *nolint.Reporter, fn *ssa.Function, sensitiveParamNames []string) {
	params := fn.Params
	for _, param := range params {
		for _, sensitiveName := range sensitiveParamNames {
			if strings.Contains(strings.ToLower(param.Name()), sensitiveName) {
				// Track where this sensitive parameter flows
				flow := TrackDataFlow(fn, param)
				for _, instr := range flow {
					if call, ok := instr.(*ssa.Call); ok {
						callee := call.Call.StaticCallee()
						if callee != nil && isLoggingCall(callee) {
							reporter.Reportf(call.Pos(),
								"sensitive parameter %q may be leaked through logging",
								param.Name())
						}
					}
				}
			}
		}
	}
}

// isLoggingCall checks if a function is a logging call
func isLoggingCall(fn *ssa.Function) bool {
	if fn.Pkg == nil {
		return false
	}

	pkgPath := fn.Pkg.Pkg.Path()
	logPackages := []string{"log", "github.com/sirupsen/logrus", "go.uber.org/zap", "github.com/rs/zerolog"}

	for _, pkg := range logPackages {
		if strings.Contains(pkgPath, pkg) {
			return true
		}
	}

	return false
}

// GetSSAPackage is a helper to get the SSA representation
func GetSSAPackage(pass *analysis.Pass) *ssa.Package {
	ssaInfo := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	return ssaInfo.Pkg
}

// GetAllFunctions returns all functions in the package
func GetAllFunctions(ssaPkg *ssa.Package) []*ssa.Function {
	var funcs []*ssa.Function

	for _, member := range ssaPkg.Members {
		if fn, ok := member.(*ssa.Function); ok {
			funcs = append(funcs, fn)
		}
		if typ, ok := member.(*ssa.Type); ok {
			// Get methods of the type
			named := typ.Type().(*types.Named)
			for i := 0; i < named.NumMethods(); i++ {
				method := named.Method(i)
				if ssaFn := ssaPkg.Prog.FuncValue(method); ssaFn != nil {
					funcs = append(funcs, ssaFn)
				}
			}
		}
	}

	return funcs
}
