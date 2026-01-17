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
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `enforce consistent component lifecycle patterns

This analyzer ensures:
1. Components with Run() also have Close() or GracefulStop()
2. Run() methods accept context.Context for cancellation
3. Long-running goroutines respect context cancellation
4. Components implement graceful shutdown patterns

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

var Analyzer = &analysis.Analyzer{
	Name:     "lifecycle",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// LifecycleMethods are methods that indicate a component has lifecycle
// Note: "Listen" is excluded because it typically follows the net.Listen() pattern
// (takes an address string and returns quickly) rather than being a blocking run method
var RunMethods = []string{"Run", "Start", "Serve"}
var StopMethods = []string{"Close", "Stop", "Shutdown", "GracefulStop", "GracefulShutdown"}

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Track types and their methods
	typeRunMethods := make(map[string]bool)   // type -> has run method
	typeStopMethods := make(map[string]bool)  // type -> has stop method
	runMethodPos := make(map[string]ast.Node) // type -> run method position

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	// First pass: collect method information
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			return
		}

		recvType := getReceiverTypeName(fn.Recv.List[0].Type)
		if recvType == "" {
			return
		}

		// Check for Run methods
		for _, runMethod := range RunMethods {
			if fn.Name.Name == runMethod {
				typeRunMethods[recvType] = true
				runMethodPos[recvType] = fn

				// Check if Run accepts context
				checkRunAcceptsContext(reporter, fn)

				// Check if Run respects context cancellation
				checkRunRespectsContext(reporter, fn)
			}
		}

		// Check for Stop methods
		for _, stopMethod := range StopMethods {
			if fn.Name.Name == stopMethod {
				typeStopMethods[recvType] = true
			}
		}
	})

	// Report types with Run but no Stop
	for typeName, hasRun := range typeRunMethods {
		if hasRun && !typeStopMethods[typeName] {
			if pos, ok := runMethodPos[typeName]; ok {
				reporter.Reportf(pos.Pos(),
					"type %q has Run() method but no Close()/Stop()/GracefulStop() method; "+
						"consider adding a method for graceful shutdown",
					typeName)
			}
		}
	}

	return nil, nil
}

// getReceiverTypeName extracts the type name from a receiver expression
func getReceiverTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		if ident, ok := e.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

// checkRunAcceptsContext verifies that Run() accepts context.Context
func checkRunAcceptsContext(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		reporter.Reportf(fn.Pos(),
			"%s() should accept context.Context as first parameter for cancellation support",
			fn.Name.Name)
		return
	}

	firstParam := fn.Type.Params.List[0]
	paramType := types.ExprString(firstParam.Type)

	if !strings.Contains(paramType, "Context") {
		reporter.Reportf(fn.Pos(),
			"%s() first parameter should be context.Context, got %s",
			fn.Name.Name, paramType)
	}
}

// checkRunRespectsContext checks if Run() has proper context handling
func checkRunRespectsContext(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}

	hasContextDoneCheck := false
	hasForLoop := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			hasForLoop = true

		case *ast.SelectStmt:
			// Check if select has ctx.Done() case
			for _, comm := range node.Body.List {
				if commClause, ok := comm.(*ast.CommClause); ok {
					if commClause.Comm != nil {
						// Check for <-ctx.Done() pattern
						if exprStmt, ok := commClause.Comm.(*ast.ExprStmt); ok {
							if unary, ok := exprStmt.X.(*ast.UnaryExpr); ok {
								commStr := types.ExprString(unary.X)
								if strings.Contains(commStr, "Done()") || strings.Contains(commStr, "ctx.Done") {
									hasContextDoneCheck = true
								}
							}
						}
					}
				}
			}
		}

		return true
	})

	// If there's a for loop without context check, warn
	if hasForLoop && !hasContextDoneCheck {
		reporter.Reportf(fn.Pos(),
			"%s() has a loop but doesn't check ctx.Done(); "+
				"consider adding select with <-ctx.Done() case for graceful shutdown",
			fn.Name.Name)
	}
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
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	typeRunMethods := make(map[string]bool)
	typeStopMethods := make(map[string]bool)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			return
		}

		recvType := getReceiverTypeName(fn.Recv.List[0].Type)
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
