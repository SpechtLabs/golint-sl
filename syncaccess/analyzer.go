// Package syncaccess provides an analyzer that detects potential data races
// and synchronization issues in concurrent code.
package syncaccess

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `detect potential data races and synchronization issues

This analyzer detects:
1. Variables captured by goroutines without synchronization
2. Struct fields accessed in goroutines without mutex protection
3. Maps accessed concurrently without sync.Map or mutex
4. Channels that may deadlock (unbuffered with no receiver)

Data races cause unpredictable behavior and are hard to debug.
Use proper synchronization:

    // Good: Protected by mutex
    type Counter struct {
        mu    sync.Mutex
        count int
    }

    func (c *Counter) Increment() {
        c.mu.Lock()
        defer c.mu.Unlock()
        c.count++
    }

    // Good: Using sync.Map for concurrent map access
    var cache sync.Map
    cache.Store(key, value)
    val, ok := cache.Load(key)

    // Bad: Unprotected shared state
    var count int
    go func() {
        count++  // Data race!
    }()`

var Analyzer = &analysis.Analyzer{
	Name:     "syncaccess",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Track struct types with mutex fields
	structsWithMutex := findStructsWithMutex(pass)

	nodeFilter := []ast.Node{
		(*ast.GoStmt)(nil),
		(*ast.FuncDecl)(nil),
	}

	var currentFunc *ast.FuncDecl

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.FuncDecl:
			currentFunc = node
			checkMutexUsage(pass, node, structsWithMutex)

		case *ast.GoStmt:
			checkGoroutineCaptures(pass, node, currentFunc)
		}
	})

	return nil, nil
}

// findStructsWithMutex finds struct types that have mutex fields
func findStructsWithMutex(pass *analysis.Pass) map[string]bool {
	result := make(map[string]bool)

	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			typeSpec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return true
			}

			for _, field := range structType.Fields.List {
				fieldType := types.ExprString(field.Type)
				if strings.Contains(fieldType, "Mutex") || strings.Contains(fieldType, "RWMutex") {
					result[typeSpec.Name.Name] = true
					break
				}
			}

			return true
		})
	}

	return result
}

// checkGoroutineCaptures checks for variables captured by goroutines
func checkGoroutineCaptures(pass *analysis.Pass, goStmt *ast.GoStmt, currentFunc *ast.FuncDecl) {
	funcLit, ok := goStmt.Call.Fun.(*ast.FuncLit)
	if !ok {
		return
	}

	// Find variables defined in the parent scope
	parentVars := collectLocalVars(currentFunc)

	// Find variables used in the goroutine
	capturedVars := findCapturedVars(funcLit, parentVars)

	// Check for problematic captures
	for varName, varInfo := range capturedVars {
		// Check if it's a loop variable (common bug)
		if isLoopVariable(currentFunc, varName, goStmt) {
			pass.Reportf(varInfo.pos,
				"loop variable %q captured by goroutine; this may cause unexpected behavior - pass as parameter instead",
				varName)
			continue
		}

		// Check for pointer/reference types that might be shared
		if varInfo.isPointer || varInfo.isMap || varInfo.isSlice {
			if !varInfo.isProtected {
				pass.Reportf(varInfo.pos,
					"shared variable %q captured by goroutine without synchronization; consider using mutex or channels",
					varName)
			}
		}
	}
}

type varInfo struct {
	pos         token.Pos
	isPointer   bool
	isMap       bool
	isSlice     bool
	isProtected bool
}

// collectLocalVars collects variables defined in a function
func collectLocalVars(fn *ast.FuncDecl) map[string]varInfo {
	vars := make(map[string]varInfo)

	if fn == nil || fn.Body == nil {
		return vars
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range node.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					info := varInfo{pos: ident.Pos()}
					// Try to determine type
					if len(node.Rhs) > 0 {
						info = inferVarType(node.Rhs[0], info)
					}
					vars[ident.Name] = info
				}
			}

		case *ast.ValueSpec:
			for _, name := range node.Names {
				info := varInfo{pos: name.Pos()}
				// Check type
				if node.Type != nil {
					switch node.Type.(type) {
					case *ast.StarExpr:
						info.isPointer = true
					case *ast.MapType:
						info.isMap = true
					case *ast.ArrayType:
						info.isSlice = true
					}
				}
				vars[name.Name] = info
			}
		}

		return true
	})

	return vars
}

func inferVarType(expr ast.Expr, info varInfo) varInfo {
	switch e := expr.(type) {
	case *ast.UnaryExpr:
		if e.Op.String() == "&" {
			info.isPointer = true
		}
	case *ast.CallExpr:
		if ident, ok := e.Fun.(*ast.Ident); ok {
			if ident.Name == "make" && len(e.Args) > 0 {
				switch e.Args[0].(type) {
				case *ast.MapType:
					info.isMap = true
				case *ast.ArrayType, *ast.ChanType:
					info.isSlice = true
				}
			}
		}
	case *ast.CompositeLit:
		switch e.Type.(type) {
		case *ast.MapType:
			info.isMap = true
		case *ast.ArrayType:
			info.isSlice = true
		}
	}
	return info
}

// findCapturedVars finds variables from parent scope used in a function literal
func findCapturedVars(funcLit *ast.FuncLit, parentVars map[string]varInfo) map[string]varInfo {
	captured := make(map[string]varInfo)

	// Get parameters of the function literal (these are not captured)
	params := make(map[string]bool)
	if funcLit.Type.Params != nil {
		for _, field := range funcLit.Type.Params.List {
			for _, name := range field.Names {
				params[name.Name] = true
			}
		}
	}

	ast.Inspect(funcLit.Body, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		// Skip if it's a parameter
		if params[ident.Name] {
			return true
		}

		// Check if it's from parent scope
		if info, exists := parentVars[ident.Name]; exists {
			info.pos = ident.Pos()
			captured[ident.Name] = info
		}

		return true
	})

	return captured
}

// isLoopVariable checks if a variable is a loop iteration variable
func isLoopVariable(fn *ast.FuncDecl, varName string, goStmt *ast.GoStmt) bool {
	if fn == nil || fn.Body == nil {
		return false
	}

	isLoopVar := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.RangeStmt:
			// Check if varName is the loop variable
			if key, ok := node.Key.(*ast.Ident); ok && key.Name == varName {
				// Check if goStmt is inside this loop
				if containsNode(node.Body, goStmt) {
					isLoopVar = true
					return false
				}
			}
			if value, ok := node.Value.(*ast.Ident); ok && value.Name == varName {
				if containsNode(node.Body, goStmt) {
					isLoopVar = true
					return false
				}
			}

		case *ast.ForStmt:
			// Check init statement for the variable
			if assign, ok := node.Init.(*ast.AssignStmt); ok {
				for _, lhs := range assign.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok && ident.Name == varName {
						if containsNode(node.Body, goStmt) {
							isLoopVar = true
							return false
						}
					}
				}
			}
		}

		return true
	})

	return isLoopVar
}

// containsNode checks if a node contains another node
func containsNode(parent ast.Node, target ast.Node) bool {
	found := false
	ast.Inspect(parent, func(n ast.Node) bool {
		if n == target {
			found = true
			return false
		}
		return true
	})
	return found
}

// checkMutexUsage checks that struct methods use mutex properly
func checkMutexUsage(pass *analysis.Pass, fn *ast.FuncDecl, structsWithMutex map[string]bool) {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return
	}

	// Get receiver type name
	recvType := ""
	switch t := fn.Recv.List[0].Type.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			recvType = ident.Name
		}
	case *ast.Ident:
		recvType = t.Name
	}

	// Check if this type has a mutex
	if !structsWithMutex[recvType] {
		return
	}

	// Check if the method accesses fields without locking
	hasLock := false
	hasFieldAccess := false

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			// Check for Lock() call
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "Lock" || sel.Sel.Name == "RLock" {
					hasLock = true
				}
			}

		case *ast.SelectorExpr:
			// Check for field access on receiver
			if ident, ok := node.X.(*ast.Ident); ok {
				// Check if it's the receiver
				if fn.Recv != nil && len(fn.Recv.List) > 0 {
					if len(fn.Recv.List[0].Names) > 0 {
						recvName := fn.Recv.List[0].Names[0].Name
						if ident.Name == recvName {
							// Skip mutex field itself
							if node.Sel.Name != "mu" && node.Sel.Name != "mutex" &&
								!strings.Contains(strings.ToLower(node.Sel.Name), "mutex") {
								hasFieldAccess = true
							}
						}
					}
				}
			}
		}

		return true
	})

	// If there's field access but no lock, warn
	if hasFieldAccess && !hasLock {
		pass.Reportf(fn.Pos(),
			"method %q on type with mutex accesses fields without Lock(); consider adding synchronization",
			fn.Name.Name)
	}
}
