// Package statusupdate provides an analyzer that ensures Kubernetes reconcilers
// properly update the Status subresource after making changes.
package statusupdate

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the analyzer's documentation.
const Doc = `ensure reconcilers update Status after changes

This analyzer detects reconcilers that:
1. Modify spec or status fields but don't call Status().Update() or
   Status().Patch()
2. Create/Update resources but don't reflect state in Status
3. Handle errors without updating Status.Conditions

Kubernetes best practice is to always update Status to reflect current state,
including error conditions. This allows users and other controllers to observe
the actual state of resources.

Only an Update, Patch or Apply through Status() or SubResource("status"),
directly or through a variable assigned one, persists the status. Update and
Patch on the object itself ignore the status subresource, so assigning
obj.Status fields and then calling client.Update leaves them unsaved. A Patch on something that isn't a client (a patch helper
such as cluster-api's patch.Helper, which has no Status method) persists spec
and status together and counts as a status update too, and so does a call of
a function or method of the package that writes the status, such as an
updateStatus helper.`

// Analyzer reports reconcilers that change resources without updating their Status.
var Analyzer = &analysis.Analyzer{
	Name:     "statusupdate",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// statusField is the name of the Status subresource field and accessor.
const statusField = "Status"

// patchMethod is the method a patch helper persists spec and status with.
const patchMethod = "Patch"

// pkgFacts holds what the analyzer learns about the whole package before it
// looks at a reconciler.
type pkgFacts struct {
	// decls maps the package's functions and methods to their declarations.
	decls map[*types.Func]*ast.FuncDecl
	// writers holds the variables assigned a status writer.
	writers map[types.Object]bool
}

// statusUsage records which mutation and status operations a reconciler performs
type statusUsage struct {
	info             *types.Info
	facts            *pkgFacts
	visited          map[*types.Func]bool
	resourceMutation bool
	statusUpdate     bool
	conditionUpdate  bool
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	facts := collectFacts(pass)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}

		if !isReconcileFunction(fn) {
			return
		}

		checkReconcilerStatus(reporter, pass.TypesInfo, facts, fn)
	})

	return nil, nil
}

// collectFacts maps the functions and methods declared in the package to
// their declarations, so that a status write in a helper counts for its
// caller, and records the variables that hold a status writer, so that
// sw := r.Status(); sw.Update(ctx, obj) counts like r.Status().Update(ctx, obj).
func collectFacts(pass *analysis.Pass) *pkgFacts {
	facts := &pkgFacts{
		decls:   make(map[*types.Func]*ast.FuncDecl),
		writers: make(map[types.Object]bool),
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncDecl:
				if fn, ok := pass.TypesInfo.Defs[n.Name].(*types.Func); ok {
					facts.decls[fn] = n
				}
			case *ast.AssignStmt:
				facts.recordWriters(pass.TypesInfo, n.Lhs, n.Rhs)
			case *ast.ValueSpec:
				names := make([]ast.Expr, len(n.Names))
				for i, name := range n.Names {
					names[i] = name
				}
				facts.recordWriters(pass.TypesInfo, names, n.Values)
			}
			return true
		})
	}

	return facts
}

// recordWriters records the variables among lhs that are assigned a call of
// Status() or SubResource("status") from the matching expression of rhs.
func (f *pkgFacts) recordWriters(info *types.Info, lhs, rhs []ast.Expr) {
	if len(lhs) != len(rhs) {
		return
	}

	for i, expr := range lhs {
		ident, ok := expr.(*ast.Ident)
		if !ok || !isStatusWriterCall(rhs[i]) {
			continue
		}
		if obj := info.ObjectOf(ident); obj != nil {
			f.writers[obj] = true
		}
	}
}

func isReconcileFunction(fn *ast.FuncDecl) bool {
	if fn.Name == nil || fn.Name.Name != "Reconcile" {
		return false
	}

	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return false
	}

	recv := fn.Recv.List[0]
	recvType := types.ExprString(recv.Type)

	patterns := []string{"Reconciler", "Controller", "Operator"}
	for _, pattern := range patterns {
		if strings.Contains(recvType, pattern) {
			return true
		}
	}

	return false
}

func checkReconcilerStatus(reporter *nolint.Reporter, info *types.Info, facts *pkgFacts, fn *ast.FuncDecl) {
	usage := statusUsage{info: info, facts: facts, visited: make(map[*types.Func]bool)}

	// Track what operations are performed
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			usage.recordCall(call)
			if !usage.statusUpdate && usage.writesStatus(call) {
				usage.statusUpdate = true
			}
		}
		return true
	})

	// Also check for direct Conditions assignments
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if assign, ok := n.(*ast.AssignStmt); ok {
			usage.recordAssignment(assign)
		}
		return true
	})

	// Report issues
	if usage.resourceMutation && !usage.statusUpdate {
		reporter.Reportf(fn.Pos(),
			"reconciler mutates resources but doesn't update Status; use Status().Update() to reflect current state")
	}

	// Only warn about missing conditions if there's complex logic
	if usage.resourceMutation && !usage.conditionUpdate && hasComplexLogic(fn) {
		reporter.Reportf(fn.Pos(),
			"reconciler performs mutations but doesn't update Status.Conditions; consider using conditions to report state")
	}
}

// recordCall records the mutation, Status and condition operations performed by a call
func (u *statusUsage) recordCall(call *ast.CallExpr) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	methodName := sel.Sel.Name

	// Check for resource mutations
	mutationMethods := []string{"Create", "Update", "Patch", "Delete"}
	for _, method := range mutationMethods {
		if methodName == method {
			u.resourceMutation = true
		}
	}

	// Check for condition updates (various patterns)
	conditionPatterns := []string{
		"SetCondition",
		"SetConditions",
		"UpdateCondition",
		"SetStatusCondition",
		"SetTypedCondition",
		"SetReadyCondition",
	}
	for _, pattern := range conditionPatterns {
		if strings.Contains(methodName, pattern) || methodName == pattern {
			u.conditionUpdate = true
		}
	}

	// Check for meta.SetStatusCondition
	if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "meta" && strings.Contains(methodName, "Condition") {
		u.conditionUpdate = true
	}
}

// recordAssignment records direct assignments to Conditions fields
func (u *statusUsage) recordAssignment(assign *ast.AssignStmt) {
	for _, lhs := range assign.Lhs {
		sel, ok := lhs.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		// Check for .Conditions assignments
		if sel.Sel.Name == "Conditions" {
			u.conditionUpdate = true
		}
	}
}

// writesStatus reports whether call persists the status, either itself or
// through a function or method declared in the package that does, such as a
// reconciler's updateStatus helper.
func (u *statusUsage) writesStatus(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok && u.persistsStatus(sel) {
		return true
	}

	callee := typeutil.StaticCallee(u.info, call)
	if callee == nil || u.visited[callee.Origin()] {
		return false
	}
	u.visited[callee.Origin()] = true

	decl := u.facts.decls[callee.Origin()]
	if decl == nil || decl.Body == nil {
		return false
	}

	found := false
	ast.Inspect(decl.Body, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok && u.writesStatus(c) {
			found = true
		}
		return !found
	})

	return found
}

// persistsStatus reports whether the method call sel writes the status: an
// Update, Patch or Apply on Status() or SubResource("status"), or a Patch on
// a patch helper. Update and Patch on the object itself don't persist the
// status, since the API server ignores status changes there when the status
// subresource is enabled.
func (u *statusUsage) persistsStatus(sel *ast.SelectorExpr) bool {
	switch sel.Sel.Name {
	case "Update", patchMethod, "Apply":
		if u.isStatusWriter(sel.X) {
			return true
		}
	}

	return sel.Sel.Name == patchMethod && u.isPatchHelper(sel.X)
}

// isStatusWriter reports whether expr is the writer for the status
// subresource: a call of Status() or SubResource("status"), or a variable
// assigned one.
func (u *statusUsage) isStatusWriter(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return u.facts.writers[u.info.ObjectOf(ident)]
	}

	return isStatusWriterCall(expr)
}

// isStatusWriterCall reports whether expr is a call of Status() or of
// SubResource("status"), which return the writer for the status subresource.
func isStatusWriterCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	switch sel.Sel.Name {
	case statusField:
		return len(call.Args) == 0
	case "SubResource":
		return len(call.Args) == 1 && isStatusLiteral(call.Args[0])
	default:
		return false
	}
}

// isStatusLiteral reports whether expr is the string literal "status".
func isStatusLiteral(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Value == `"status"`
}

// isPatchHelper reports whether the receiver expr is a value, not a package,
// whose type has no Status method: a patch helper such as cluster-api's
// patch.Helper, which persists spec and status together. A Kubernetes client
// has a Status method, and its Patch leaves the status alone.
func (u *statusUsage) isPatchHelper(expr ast.Expr) bool {
	// A package name has no entry in Types: pkg.Patch is a function call.
	tv, ok := u.info.Types[expr]
	if !ok || !tv.IsValue() {
		return false
	}

	obj, _, _ := types.LookupFieldOrMethod(tv.Type, true, nil, statusField)
	_, hasStatusMethod := obj.(*types.Func)

	return !hasStatusMethod
}

func hasComplexLogic(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}

	// Count complexity indicators
	complexity := 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt, *ast.RangeStmt:
			complexity++
		case *ast.SwitchStmt, *ast.TypeSwitchStmt:
			complexity++
		case *ast.SelectStmt:
			complexity++
		}
		return complexity < 5
	})

	return complexity >= 3
}
