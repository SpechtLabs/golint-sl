// Package syncaccess provides an analyzer that detects potential data races
// and synchronization issues in concurrent code.
package syncaccess

import (
	"go/ast"
	"go/token"
	"go/types"
	"go/version"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the syncaccess analyzer's documentation.
const Doc = `detect potential data races and synchronization issues

This analyzer detects:
1. Variables captured by goroutines without synchronization: a map,
   slice, array or pointer the enclosing function declared, used by a
   goroutine that doesn't synchronize; a local variable or parameter a
   goroutine that doesn't synchronize assigns (or, for a map, writes an
   element of) while the enclosing function still uses it after the go
   statement; and a map the goroutines of a loop write without a lock.
   A goroutine synchronizes when it takes a lock, uses a channel, calls
   into sync or sync/atomic, or calls a function value or an interface
2. Loop variables shared by every iteration and captured by a goroutine;
   from Go 1.22 on, a variable the for clause declares is per iteration
   and is not reported
3. Methods of a struct with a mutex field that access the struct's
   fields without calling Lock or RLock

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
    }()
    fmt.Println(count)`

// perIterationLoopVars is the first language version in which a variable
// declared by a for clause is a new variable in every iteration.
const perIterationLoopVars = "go1.22"

// Analyzer reports potential data races and synchronization issues.
var Analyzer = &analysis.Analyzer{
	Name:     "syncaccess",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// capture is a local variable of the enclosing function that a goroutine's
// function literal uses.
type capture struct {
	obj      *types.Var
	pos      token.Pos // the first use inside the function literal
	written  bool      // the goroutine assigns the variable itself
	mapWrite bool      // the goroutine writes or deletes an element of the map
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Track struct types with mutex fields
	structsWithMutex := findStructsWithMutex(pass)

	// Local variables the program declares as maps, slices or pointers
	references := collectReferenceVars(pass, insp)

	insp.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
		checkMutexUsage(reporter, n.(*ast.FuncDecl), structsWithMutex)
	})

	insp.WithStack([]ast.Node{(*ast.GoStmt)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if push {
			checkGoroutineCaptures(pass, reporter, n.(*ast.GoStmt), stack, references)
		}
		return true
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

// collectReferenceVars returns the variables whose declaration makes them a
// map, slice, array or pointer: a var declaration with an explicit type, or a
// short variable declaration that initializes them with make, a composite
// literal or an address. Later assignments don't change the result.
func collectReferenceVars(pass *analysis.Pass, insp *inspector.Inspector) map[types.Object]bool {
	refs := make(map[types.Object]bool)

	nodeFilter := []ast.Node{
		(*ast.AssignStmt)(nil),
		(*ast.ValueSpec)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.AssignStmt:
			if node.Tok != token.DEFINE || len(node.Lhs) != len(node.Rhs) {
				return
			}
			for i, lhs := range node.Lhs {
				// The left-hand side of := holds identifiers only. A
				// redeclared variable has no definition here and keeps the
				// kind of its first declaration.
				if obj := pass.TypesInfo.Defs[lhs.(*ast.Ident)]; obj != nil && isReferenceExpr(pass, node.Rhs[i]) {
					refs[obj] = true
				}
			}

		case *ast.ValueSpec:
			// Only an explicit type counts here: a var declaration with an
			// initializer and no type is left alone.
			if node.Type == nil || !isReferenceTypeExpr(node.Type) {
				return
			}
			for _, name := range node.Names {
				// The blank identifier has no object, so it's never captured.
				refs[pass.TypesInfo.Defs[name]] = true
			}
		}
	})

	return refs
}

// isReferenceTypeExpr reports whether a declared type is a pointer, map,
// slice or array type.
func isReferenceTypeExpr(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.StarExpr, *ast.MapType, *ast.ArrayType:
		return true
	default:
		return false
	}
}

// isReferenceExpr reports whether an initializer creates a pointer, map,
// slice or array: an address, a make of a map or slice, or a map, slice or
// array literal.
func isReferenceExpr(pass *analysis.Pass, expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.UnaryExpr:
		return e.Op == token.AND
	case *ast.CallExpr:
		ident, ok := e.Fun.(*ast.Ident)
		if !ok || len(e.Args) == 0 {
			return false
		}
		if builtin, ok := pass.TypesInfo.Uses[ident].(*types.Builtin); !ok || builtin.Name() != "make" {
			return false
		}
		switch e.Args[0].(type) {
		case *ast.MapType, *ast.ArrayType:
			return true
		}
	case *ast.CompositeLit:
		switch e.Type.(type) {
		case *ast.MapType, *ast.ArrayType:
			return true
		}
	}

	return false
}

// checkGoroutineCaptures checks the local variables a goroutine's function
// literal captures. stack holds the go statement's ancestors, from the file
// down to the go statement itself.
func checkGoroutineCaptures(pass *analysis.Pass, reporter *nolint.Reporter, goStmt *ast.GoStmt, stack []ast.Node, references map[types.Object]bool) {
	funcLit, ok := goStmt.Call.Fun.(*ast.FuncLit)
	if !ok {
		return
	}

	captures := findCapturedVars(pass, funcLit)
	if len(captures) == 0 {
		return
	}

	file := stack[0].(*ast.File)
	decl := stack[1]
	loops := enclosingLoops(stack, goStmt)
	locks, syncs := goroutineSynchronization(pass, funcLit)

	for _, c := range captures {
		// Skip channels - they are inherently thread-safe in Go
		if _, ok := types.Unalias(c.obj.Type()).Underlying().(*types.Chan); ok {
			continue
		}

		// A variable every iteration of an enclosing loop shares
		if isSharedLoopVar(pass, file, loops, c.obj) {
			reporter.Reportf(c.pos,
				"loop variable %q captured by goroutine; this may cause unexpected behavior - pass as parameter instead",
				c.obj.Name())
			continue
		}

		// The goroutine takes a lock around its accesses
		if locks {
			continue
		}

		// The goroutines a loop starts all write the same map, whatever else
		// they synchronize with. Otherwise, a goroutine that synchronizes
		// with the function some way may well be waited for. One that
		// doesn't shares a variable the function declared as a reference as
		// soon as it uses it, and races on any other variable it writes when
		// the function uses it afterwards or the go statement runs again in
		// a loop the variable outlives.
		outlives := outlivesLoop(loops, c.obj)
		racy := c.mapWrite && outlives ||
			!syncs && (references[c.obj] ||
				(c.written || c.mapWrite) && (outlives || usedAfter(pass, decl, goStmt, c.obj)))
		if racy {
			reporter.Reportf(c.pos,
				"shared variable %q captured by goroutine without synchronization; consider using mutex or channels",
				c.obj.Name())
		}
	}
}

// findCapturedVars returns the local variables of the enclosing function
// that funcLit uses, in the order of their first use, and how the function
// literal writes them.
func findCapturedVars(pass *analysis.Pass, funcLit *ast.FuncLit) []*capture {
	byObj := make(map[*types.Var]*capture)
	var captures []*capture

	captured := func(expr ast.Expr) *capture {
		ident, ok := ast.Unparen(expr).(*ast.Ident)
		if !ok {
			return nil
		}
		obj, ok := pass.TypesInfo.Uses[ident].(*types.Var)
		if !ok {
			return nil
		}
		return byObj[obj]
	}

	markWrite := func(lhs ast.Expr) {
		if c := captured(lhs); c != nil {
			c.written = true
			return
		}
		if index, ok := ast.Unparen(lhs).(*ast.IndexExpr); ok {
			if c := captured(index.X); c != nil && isMap(c.obj) {
				c.mapWrite = true
			}
		}
	}

	// First pass: the captured variables, in the order of their first use
	ast.Inspect(funcLit.Body, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		obj, ok := pass.TypesInfo.Uses[ident].(*types.Var)
		if !ok || byObj[obj] != nil || !isCapturedLocal(pass, funcLit, obj) {
			return true
		}
		c := &capture{obj: obj, pos: ident.Pos()}
		byObj[obj] = c
		captures = append(captures, c)
		return true
	})

	// Second pass: how the function literal writes them
	ast.Inspect(funcLit.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			for _, lhs := range node.Lhs {
				markWrite(lhs)
			}
		case *ast.IncDecStmt:
			markWrite(node.X)
		case *ast.RangeStmt:
			if node.Tok == token.ASSIGN {
				for _, e := range []ast.Expr{node.Key, node.Value} {
					if e != nil {
						markWrite(e)
					}
				}
			}
		case *ast.CallExpr:
			if isBuiltinCall(pass, node, "delete") && len(node.Args) > 0 {
				if c := captured(node.Args[0]); c != nil {
					c.mapWrite = true
				}
			}
		}
		return true
	})

	return captures
}

// isCapturedLocal reports whether obj is a local variable or parameter of a
// function enclosing funcLit, declared outside funcLit.
func isCapturedLocal(pass *analysis.Pass, funcLit *ast.FuncLit, obj *types.Var) bool {
	if obj.IsField() || obj.Pkg() != pass.Pkg {
		return false
	}
	if obj.Parent() == nil || obj.Parent() == pass.Pkg.Scope() {
		return false
	}
	return obj.Pos() < funcLit.Pos() || obj.Pos() >= funcLit.End()
}

// isMap reports whether v is a map.
func isMap(v *types.Var) bool {
	_, ok := types.Unalias(v.Type()).Underlying().(*types.Map)
	return ok
}

// isBuiltinCall reports whether call calls the named builtin function.
func isBuiltinCall(pass *analysis.Pass, call *ast.CallExpr, name string) bool {
	ident, ok := ast.Unparen(call.Fun).(*ast.Ident)
	if !ok {
		return false
	}
	builtin, ok := pass.TypesInfo.Uses[ident].(*types.Builtin)
	return ok && builtin.Name() == name
}

// goroutineSynchronization reports whether funcLit takes a lock (calls Lock
// or RLock of a sync type or of a sync.Locker), and whether it may
// synchronize with the goroutine that started it in any other way: a channel
// operation, any other call into sync or sync/atomic, or a call through a
// function value or an interface, which could do either.
func goroutineSynchronization(pass *analysis.Pass, funcLit *ast.FuncLit) (locks, syncs bool) {
	ast.Inspect(funcLit.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SendStmt, *ast.SelectStmt:
			syncs = true
		case *ast.UnaryExpr:
			if node.Op == token.ARROW {
				syncs = true
			}
		case *ast.RangeStmt:
			if _, ok := types.Unalias(pass.TypesInfo.TypeOf(node.X)).Underlying().(*types.Chan); ok {
				syncs = true
			}
		case *ast.CallExpr:
			lock, sync := callSynchronization(pass, node)
			locks = locks || lock
			syncs = syncs || sync
		}
		return true
	})

	return locks, syncs
}

// callSynchronization classifies a call for goroutineSynchronization.
func callSynchronization(pass *analysis.Pass, call *ast.CallExpr) (lock, sync bool) {
	if tv, ok := pass.TypesInfo.Types[call.Fun]; ok && tv.IsType() {
		return false, false // a conversion
	}
	if _, ok := ast.Unparen(call.Fun).(*ast.FuncLit); ok {
		return false, false // goroutineSynchronization inspects its body
	}
	if isBuiltinCall(pass, call, "close") {
		return false, true
	}
	if _, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Builtin); ok {
		return false, false
	}

	fn := typeutil.StaticCallee(pass.TypesInfo, call)
	if fn == nil {
		// A function value or an interface method. Lock and RLock of a
		// sync.Locker or a sync.RWMutex's RLocker count as locking.
		if method, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func); ok && isLockMethod(method) {
			return true, true
		}
		return false, true
	}

	// Only universe objects have no package, and none is a static callee.
	switch fn.Pkg().Path() {
	case "sync":
		return isLockMethod(fn), true
	case "sync/atomic":
		return false, true
	}

	return false, false
}

// isLockMethod reports whether fn is a method named Lock or RLock.
func isLockMethod(fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	return ok && sig.Recv() != nil && (fn.Name() == "Lock" || fn.Name() == "RLock")
}

// enclosingLoops returns the for and range statements whose body contains
// goStmt, innermost last.
func enclosingLoops(stack []ast.Node, goStmt *ast.GoStmt) []ast.Node {
	var loops []ast.Node
	for _, n := range stack {
		var body *ast.BlockStmt
		switch loop := n.(type) {
		case *ast.ForStmt:
			body = loop.Body
		case *ast.RangeStmt:
			body = loop.Body
		default:
			continue
		}
		if body.Pos() <= goStmt.Pos() && goStmt.End() <= body.End() {
			loops = append(loops, n)
		}
	}
	return loops
}

// isSharedLoopVar reports whether obj is the iteration variable of one of
// loops and all iterations share it: the loop assigns a variable declared
// outside it, or the file's Go version predates per-iteration loop
// variables.
func isSharedLoopVar(pass *analysis.Pass, file *ast.File, loops []ast.Node, obj *types.Var) bool {
	for _, n := range loops {
		var tok token.Token
		var vars []ast.Expr
		switch loop := n.(type) {
		case *ast.RangeStmt:
			tok, vars = loop.Tok, []ast.Expr{loop.Key, loop.Value}
		case *ast.ForStmt:
			assign, ok := loop.Init.(*ast.AssignStmt)
			if !ok {
				continue
			}
			tok, vars = assign.Tok, assign.Lhs
		}

		for _, e := range vars {
			ident, ok := e.(*ast.Ident)
			if !ok {
				continue
			}
			switch {
			case tok == token.DEFINE && pass.TypesInfo.Defs[ident] == obj:
				return !perIteration(pass, file)
			case tok == token.ASSIGN && pass.TypesInfo.Uses[ident] == obj:
				return true
			}
		}
	}

	return false
}

// perIteration reports whether a for clause in file declares a new
// variable in every iteration. An unknown version is the toolchain's own.
func perIteration(pass *analysis.Pass, file *ast.File) bool {
	v := pass.TypesInfo.FileVersions[file]
	if v == "" {
		v = pass.Pkg.GoVersion()
	}
	return !version.IsValid(v) || version.Compare(v, perIterationLoopVars) >= 0
}

// usedAfter reports whether decl uses obj after the go statement, where the
// use can run concurrently with the goroutine. Uses earlier in an enclosing
// loop are covered by outlivesLoop.
func usedAfter(pass *analysis.Pass, decl ast.Node, goStmt *ast.GoStmt, obj *types.Var) bool {
	found := false

	ast.Inspect(decl, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok && ident.Pos() >= goStmt.End() && pass.TypesInfo.Uses[ident] == obj {
			found = true
		}
		return !found
	})

	return found
}

// outlivesLoop reports whether obj was declared before one of loops, so
// that the goroutines the loop's iterations start all share it.
func outlivesLoop(loops []ast.Node, obj *types.Var) bool {
	return slices.ContainsFunc(loops, func(loop ast.Node) bool {
		return obj.Pos() < loop.Pos()
	})
}

// checkMutexUsage checks that struct methods use mutex properly
func checkMutexUsage(reporter *nolint.Reporter, fn *ast.FuncDecl, structsWithMutex map[string]bool) {
	// A method without a body (implemented in assembly) has nothing to check.
	if fn.Recv == nil || len(fn.Recv.List) == 0 || fn.Body == nil {
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
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "Lock" || sel.Sel.Name == "RLock") {
				hasLock = true
			}

		case *ast.SelectorExpr:
			// Check for field access on receiver, skipping the mutex field itself
			if ident, ok := node.X.(*ast.Ident); ok && isReceiver(fn, ident) && !isMutexFieldName(node.Sel.Name) {
				hasFieldAccess = true
			}
		}

		return true
	})

	// If there's field access but no lock, warn
	if hasFieldAccess && !hasLock {
		reporter.Reportf(fn.Pos(),
			"method %q on type with mutex accesses fields without Lock(); consider adding synchronization",
			fn.Name.Name)
	}
}

// isReceiver reports whether ident refers to the method's named receiver.
func isReceiver(fn *ast.FuncDecl, ident *ast.Ident) bool {
	return fn.Recv != nil && len(fn.Recv.List) > 0 &&
		len(fn.Recv.List[0].Names) > 0 &&
		ident.Name == fn.Recv.List[0].Names[0].Name
}

// isMutexFieldName reports whether a field name looks like a mutex field.
func isMutexFieldName(name string) bool {
	return name == "mu" || name == "mutex" || strings.Contains(strings.ToLower(name), "mutex")
}
