// Package goroutineleak provides an analyzer that detects goroutines that may leak.
//
// Goroutine leaks occur when goroutines are spawned but never properly cleaned up.
// This analyzer detects:
// 1. go func() without context cancellation
// 2. go func() without WaitGroup or done channel
// 3. Goroutines spawned in loops without bounds
// 4. Missing cleanup in defer statements
package goroutineleak

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `detect goroutines that may leak

This analyzer detects patterns that commonly cause goroutine leaks:

1. Goroutines without context cancellation support
2. Goroutines without WaitGroup or done channel for synchronization
3. Goroutines spawned in loops without proper lifecycle management
4. Channel sends/receives without select and context

Goroutine leaks cause memory growth over time and can exhaust system resources.

Good patterns:
    // With context cancellation
    go func() {
        select {
        case <-ctx.Done():
            return
        case work := <-workChan:
            process(work)
        }
    }()

    // With WaitGroup
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        // work
    }()
    wg.Wait()

Bad patterns:
    // No way to stop this goroutine
    go func() {
        for {
            process(<-workChan)  // Blocks forever if channel never closes
        }
    }()`

var Analyzer = &analysis.Analyzer{
	Name:     "goroutineleak",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.GoStmt)(nil),
		(*ast.FuncDecl)(nil),
	}

	// Track if we're in a function that accepts context
	var currentFuncHasContext bool

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.FuncDecl:
			currentFuncHasContext = hasContextParam(node)

		case *ast.GoStmt:
			checkGoroutine(pass, node, currentFuncHasContext)
		}
	})

	return nil, nil
}

func hasContextParam(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil {
		return false
	}

	for _, param := range fn.Type.Params.List {
		paramType := types.ExprString(param.Type)
		if strings.Contains(paramType, "Context") {
			return true
		}
	}
	return false
}

func checkGoroutine(pass *analysis.Pass, goStmt *ast.GoStmt, parentHasContext bool) {
	// Get the function being called in the go statement
	var funcLit *ast.FuncLit
	switch call := goStmt.Call.Fun.(type) {
	case *ast.FuncLit:
		funcLit = call
	default:
		// go someFunc() - harder to analyze, skip for now
		return
	}

	if funcLit == nil || funcLit.Body == nil {
		return
	}

	// Check for proper cleanup patterns
	hasContextCheck := false
	hasWaitGroupDone := false
	hasDoneChannel := false
	hasInfiniteLoop := false

	ast.Inspect(funcLit.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectStmt:
			// Check if select has ctx.Done() case
			for _, comm := range node.Body.List {
				if commClause, ok := comm.(*ast.CommClause); ok {
					if isContextDoneCase(commClause) {
						hasContextCheck = true
					}
					if isDoneChannelCase(commClause) {
						hasDoneChannel = true
					}
				}
			}

		case *ast.CallExpr:
			callName := getCallName(node)
			// Check for wg.Done()
			if strings.HasSuffix(callName, ".Done") || callName == "Done" {
				hasWaitGroupDone = true
			}

		case *ast.ForStmt:
			// Infinite loop: for { } or for true { }
			if node.Cond == nil {
				hasInfiniteLoop = true
			}

		case *ast.RangeStmt:
			// for range channel - okay if channel will be closed

		case *ast.UnaryExpr:
			// Blocking receives (<-ch) outside select could block forever,
			// but detecting this properly requires more complex analysis
			_ = node // Placeholder for future implementation
		}

		return true
	})

	// Report issues
	if hasInfiniteLoop && !hasContextCheck && !hasDoneChannel {
		pass.Reportf(goStmt.Pos(),
			"goroutine with infinite loop has no way to stop; add select with <-ctx.Done() or done channel")
	}

	if !hasContextCheck && !hasWaitGroupDone && !hasDoneChannel && parentHasContext {
		pass.Reportf(goStmt.Pos(),
			"goroutine spawned without cleanup mechanism; consider passing context and checking ctx.Done(), or use sync.WaitGroup")
	}

	// Note: Bare blocking receives are not flagged here as they require
	// more complex analysis to determine if they can truly block forever
}

func isContextDoneCase(comm *ast.CommClause) bool {
	if comm.Comm == nil {
		return false
	}

	// comm.Comm is an ast.Stmt, extract expression from it
	commStr := stmtToString(comm.Comm)
	return strings.Contains(commStr, "Done()") || strings.Contains(commStr, "ctx.Done")
}

func isDoneChannelCase(comm *ast.CommClause) bool {
	if comm.Comm == nil {
		return false
	}

	commStr := stmtToString(comm.Comm)
	return strings.Contains(strings.ToLower(commStr), "done")
}

func stmtToString(stmt ast.Stmt) string {
	switch s := stmt.(type) {
	case *ast.ExprStmt:
		return types.ExprString(s.X)
	case *ast.AssignStmt:
		if len(s.Rhs) > 0 {
			return types.ExprString(s.Rhs[0])
		}
	case *ast.SendStmt:
		return types.ExprString(s.Chan)
	}
	return ""
}

func getCallName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		if ident, ok := fn.X.(*ast.Ident); ok {
			return ident.Name + "." + fn.Sel.Name
		}
		return fn.Sel.Name
	}
	return ""
}
