// Package lifecycle provides an analyzer that enforces proper component lifecycle patterns.
//
// Inspired by the compute-blade-agent patterns:
//
//	type Component interface {
//	    Run(ctx context.Context) error
//	    Close() error  // or GracefulStop(ctx context.Context) error
//	}
//
// This ensures components have consistent lifecycle management.
package lifecycle

import (
	"go/ast"
	"go/types"
	"slices"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the analyzer's documentation.
const Doc = `enforce consistent component lifecycle patterns

For every type with a run method (Run, Start or Serve), this analyzer
ensures:
1. The type also has a stop method (Close, Stop, Shutdown, GracefulStop or
   GracefulShutdown)
2. The run method takes a context.Context (or a type that implements it)
   as its first parameter
3. A run method with a long-running loop (for {} without a condition, or
   a range over a channel) observes cancellation somewhere in its body:
   by receiving from or selecting on a Done() channel, or by checking
   ctx.Err()

The lifecycle pattern ensures:
- Clean startup and shutdown
- Proper resource cleanup
- Graceful handling of termination signals

Example of good patterns:

    type Server interface {
        Run(ctx context.Context) error
        GracefulStop(ctx context.Context) error
    }

    func (s *server) Run(ctx context.Context) error {
        for {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case event := <-s.events:
                s.handle(event)
            }
        }
    }`

// Analyzer reports components whose lifecycle methods are inconsistent.
var Analyzer = &analysis.Analyzer{
	Name:     "lifecycle",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// RunMethods are methods that indicate a component has lifecycle
// Note: "Listen" is excluded because it typically follows the net.Listen() pattern
// (takes an address string and returns quickly) rather than being a blocking run method
var RunMethods = []string{"Run", "Start", "Serve"}

// StopMethods are methods that shut a lifecycle component down
var StopMethods = []string{"Close", "Stop", "Shutdown", "GracefulStop", "GracefulShutdown"}

// contextMethods are the methods of context.Context; a type with all of
// them can stand in for a context.
var contextMethods = []string{"Deadline", "Done", "Err", "Value"}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Track types and their methods, in declaration order of the run methods
	var runTypes []string
	runMethod := make(map[string]*ast.FuncDecl) // type -> its first run method
	typeStopMethods := make(map[string]bool)    // type -> has stop method

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	// First pass: collect method information
	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		recvType := receiverTypeName(pass, fn)
		if recvType == "" {
			return
		}

		if slices.Contains(RunMethods, fn.Name.Name) {
			if _, seen := runMethod[recvType]; !seen {
				runTypes = append(runTypes, recvType)
				runMethod[recvType] = fn
			}

			// Check if Run accepts context
			checkRunAcceptsContext(pass, reporter, fn)

			// Check if Run respects context cancellation
			checkRunRespectsContext(pass, reporter, fn)
		}

		if slices.Contains(StopMethods, fn.Name.Name) {
			typeStopMethods[recvType] = true
		}
	})

	// Report types with Run but no Stop
	for _, typeName := range runTypes {
		if typeStopMethods[typeName] {
			continue
		}
		fn := runMethod[typeName]
		reporter.Reportf(fn.Pos(),
			"type %q has %s() method but no Close()/Stop()/GracefulStop() method; "+
				"consider adding a method for graceful shutdown",
			typeName, fn.Name.Name)
	}

	return nil, nil
}

// receiverTypeName returns the name of the type fn is a method of, with the
// pointer and any type parameters stripped, or "" when fn is not a method.
func receiverTypeName(pass *analysis.Pass, fn *ast.FuncDecl) string {
	if fn.Recv == nil {
		return ""
	}
	obj, ok := pass.TypesInfo.Defs[fn.Name].(*types.Func)
	if !ok {
		return ""
	}
	recv := obj.Signature().Recv()
	if recv == nil {
		return ""
	}

	t := types.Unalias(recv.Type())
	if ptr, ok := t.(*types.Pointer); ok {
		t = types.Unalias(ptr.Elem())
	}
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name()
	}
	return ""
}

// checkRunAcceptsContext verifies that Run() accepts context.Context
func checkRunAcceptsContext(pass *analysis.Pass, reporter *nolint.Reporter, fn *ast.FuncDecl) {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		reporter.Reportf(fn.Pos(),
			"%s() should accept context.Context as first parameter for cancellation support",
			fn.Name.Name)
		return
	}

	firstParam := fn.Type.Params.List[0]
	if !isContext(pass.TypesInfo.TypeOf(firstParam.Type)) {
		reporter.Reportf(fn.Pos(),
			"%s() first parameter should be context.Context, got %s",
			fn.Name.Name, types.ExprString(firstParam.Type))
	}
}

// isContext reports whether t is context.Context or a type that implements
// it, such as a framework's request context.
func isContext(t types.Type) bool {
	if t == nil {
		return false
	}
	if named, ok := types.Unalias(t).(*types.Named); ok {
		obj := named.Obj()
		if obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context" {
			return true
		}
	}
	for _, name := range contextMethods {
		if obj, _, _ := types.LookupFieldOrMethod(t, true, nil, name); !isFunc(obj) {
			return false
		}
	}
	return true
}

// checkRunRespectsContext checks if Run() has proper context handling
func checkRunRespectsContext(pass *analysis.Pass, reporter *nolint.Reporter, fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}

	hasContextDoneCheck := false
	hasLongRunningLoop := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ForStmt:
			// for {} runs until something breaks out of it; a loop with a
			// condition is bounded by that condition
			if node.Cond == nil {
				hasLongRunningLoop = true
			}

		case *ast.RangeStmt:
			// Ranging over a channel lasts as long as the sender; ranging
			// over a slice, map, string or integer is bounded
			if isChan(pass.TypesInfo.TypeOf(node.X)) {
				hasLongRunningLoop = true
			}

		case *ast.CallExpr:
			if observesCancellation(pass, node) {
				hasContextDoneCheck = true
			}
		}

		return true
	})

	// If there's a long-running loop without a context check, warn
	if hasLongRunningLoop && !hasContextDoneCheck {
		reporter.Reportf(fn.Pos(),
			"%s() has a loop but doesn't check ctx.Done(); "+
				"consider adding select with <-ctx.Done() case for graceful shutdown",
			fn.Name.Name)
	}
}

// observesCancellation reports whether call is a Done() call that returns a
// channel (ctx.Done(), or a component's own done channel), or ctx.Err() on
// a context.
func observesCancellation(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}

	switch sel.Sel.Name {
	case "Done":
		return isChan(pass.TypesInfo.TypeOf(call))

	case "Err":
		return isContext(pass.TypesInfo.TypeOf(sel.X))
	}
	return false
}

// isFunc reports whether obj is a function or method.
func isFunc(obj types.Object) bool {
	_, ok := obj.(*types.Func)
	return ok
}

// isChan reports whether t is a channel type.
func isChan(t types.Type) bool {
	if t == nil {
		return false
	}
	_, ok := t.Underlying().(*types.Chan)
	return ok
}

// LifecycleInfo contains information about lifecycle patterns
type LifecycleInfo struct {
	TypesWithRun     []string
	TypesWithStop    []string
	TypesMissingStop []string
	TypesWithBothRun []string // Types that have both Run and Stop
}

// AnalyzeLifecycle returns information about lifecycle patterns
func AnalyzeLifecycle(pass *analysis.Pass) *LifecycleInfo {
	info := &LifecycleInfo{}
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	typeRunMethods := make(map[string]bool)
	typeStopMethods := make(map[string]bool)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return
		}

		recvType := receiverTypeName(pass, fn)
		if recvType == "" {
			return
		}

		for _, runMethod := range RunMethods {
			if fn.Name.Name == runMethod {
				typeRunMethods[recvType] = true
			}
		}

		for _, stopMethod := range StopMethods {
			if fn.Name.Name == stopMethod {
				typeStopMethods[recvType] = true
			}
		}
	})

	for typeName := range typeRunMethods {
		info.TypesWithRun = append(info.TypesWithRun, typeName)
		if typeStopMethods[typeName] {
			info.TypesWithBothRun = append(info.TypesWithBothRun, typeName)
		} else {
			info.TypesMissingStop = append(info.TypesMissingStop, typeName)
		}
	}

	for typeName := range typeStopMethods {
		info.TypesWithStop = append(info.TypesWithStop, typeName)
	}

	return info
}
