// Package errorwrap provides an analyzer that detects bare error returns
// without proper context wrapping.
package errorwrap

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the errorwrap analyzer's documentation.
const Doc = `detect bare error returns without context

This analyzer detects:
1. Returning err directly without wrapping (return err)
2. Error returns in functions with multiple operations where context is lost
3. Error variables returned without adding context about what failed

A returned error variable counts as wrapped when the assignment that last
set it before the return wraps it (err = fmt.Errorf("...: %w", err)), when
an assignment between that one and the return wraps it, or, for a named
result, when a deferred function wraps it. A naked return inside
"if err != nil" counts as returning the named result err.

Errors should be wrapped with context to create a clear error chain:
  return humane.Wrap(err, "failed to create user", "check database connection")
  return fmt.Errorf("failed to create user: %w", err)

Prefer humane.Wrap() as it provides actionable advice to users.
Bare error returns make debugging difficult because you lose the stack context.`

// Analyzer reports error returns that lack wrapping context.
var Analyzer = &analysis.Analyzer{
	Name:     "errorwrap",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// errVarName is the conventional name of an error variable.
const errVarName = "err"

// assignment is one write to a variable inside the checked function.
type assignment struct {
	path  []ast.Node // ancestors of the write, outermost first
	pos   token.Pos
	wraps bool // whether the assigned value is a wrapping call
	clean bool // whether the assigned value is a wrapping call or nil
}

// returnSite is a return statement together with the signature of the
// function it returns from.
type returnSite struct {
	ret     *ast.ReturnStmt
	results *ast.FieldList
	// nonNil holds the variables that an enclosing "if v != nil" guarantees
	// to be non-nil at the return.
	nonNil map[types.Object]bool
	path   []ast.Node // ancestors of the return, outermost first
}

// bodyFacts is what checkFunction learns from one walk over a function body.
type bodyFacts struct {
	// assigned holds the names of variables assigned in the body that look
	// like error variables.
	assigned map[string]bool
	// writes holds every write to a variable, in source order.
	writes map[types.Object][]assignment
	// deferredWrap holds the named results a deferred function wraps.
	deferredWrap map[types.Object]bool
	returns      []returnSite
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}

		// Skip test functions
		if fn.Name != nil && strings.HasPrefix(fn.Name.Name, "Test") {
			return
		}

		// Skip very simple functions (1-2 statements)
		if len(fn.Body.List) <= 2 {
			return
		}

		checkFunction(pass, reporter, fn)
	})

	return nil, nil
}

func checkFunction(pass *analysis.Pass, reporter *nolint.Reporter, fn *ast.FuncDecl) {
	// Skip if the function returns humane.Error - these are already wrapped
	if returnsHumaneError(fn) {
		return
	}

	facts := collectFacts(pass, fn)
	for _, site := range facts.returns {
		for _, ident := range returnedIdents(pass, site) {
			checkReturnedIdent(pass, reporter, fn, site, ident, facts)
		}
	}
}

// collectFacts walks the function body once, recording the error-looking
// assignments, every write to a variable with the block it happens in, the
// named results wrapped by deferred functions, and the return statements.
func collectFacts(pass *analysis.Pass, fn *ast.FuncDecl) *bodyFacts {
	facts := &bodyFacts{
		assigned:     make(map[string]bool),
		writes:       make(map[types.Object][]assignment),
		deferredWrap: make(map[types.Object]bool),
	}

	resultObjs := make(map[types.Object]bool)
	recordNamedResults(pass, facts, resultObjs, fn.Type.Results, fn.Body)
	funcResults := []*ast.FieldList{fn.Type.Results}
	var stack []ast.Node

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if n == nil {
			if _, ok := stack[len(stack)-1].(*ast.FuncLit); ok {
				funcResults = funcResults[:len(funcResults)-1]
			}
			stack = stack[:len(stack)-1]
			return true
		}

		switch node := n.(type) {
		case *ast.FuncLit:
			funcResults = append(funcResults, node.Type.Results)
			recordNamedResults(pass, facts, resultObjs, node.Type.Results, node.Body)

		case *ast.AssignStmt:
			trackErrorNames(node.Lhs, facts.assigned)
			recordWrites(pass, facts, stack, resultObjs, node.Pos(), node.Lhs, node.Rhs)

		case *ast.ValueSpec:
			lhs := make([]ast.Expr, len(node.Names))
			for i, name := range node.Names {
				lhs[i] = name
			}
			recordWrites(pass, facts, stack, resultObjs, node.Pos(), lhs, node.Values)

		case *ast.DeferStmt:
			recordDeferredHelper(pass, facts, resultObjs, node.Call)

		case *ast.ReturnStmt:
			facts.returns = append(facts.returns, returnSite{
				ret:     node,
				results: funcResults[len(funcResults)-1],
				nonNil:  nonNilGuards(pass, stack),
				path:    slices.Clone(stack),
			})
		}

		stack = append(stack, n)
		return true
	})

	return facts
}

// recordNamedResults records a signature's named results, each with the
// write of its zero value, nil, at the start of body.
func recordNamedResults(pass *analysis.Pass, facts *bodyFacts, resultObjs map[types.Object]bool, results *ast.FieldList, body *ast.BlockStmt) {
	if results == nil {
		return
	}
	for _, field := range results.List {
		for _, name := range field.Names {
			obj := pass.TypesInfo.ObjectOf(name)
			if obj == nil {
				continue
			}
			resultObjs[obj] = true
			facts.writes[obj] = append(facts.writes[obj], assignment{path: []ast.Node{body}, pos: name.Pos(), clean: true})
		}
	}
}

// recordDeferredHelper records the named results whose address a deferred
// call receives, as in "defer wrapErr(&err, ...)": the helper is taken to
// wrap them.
func recordDeferredHelper(pass *analysis.Pass, facts *bodyFacts, resultObjs map[types.Object]bool, call *ast.CallExpr) {
	for _, arg := range call.Args {
		unary, ok := ast.Unparen(arg).(*ast.UnaryExpr)
		if !ok || unary.Op != token.AND {
			continue
		}
		ident, ok := ast.Unparen(unary.X).(*ast.Ident)
		if !ok {
			continue
		}
		if obj := pass.TypesInfo.ObjectOf(ident); obj != nil && resultObjs[obj] {
			facts.deferredWrap[obj] = true
		}
	}
}

// recordWrites records the writes of one assignment or var declaration.
// stack holds the ancestors of the statement, outermost first.
func recordWrites(pass *analysis.Pass, facts *bodyFacts, stack []ast.Node, resultObjs map[types.Object]bool, pos token.Pos, lhs, rhs []ast.Expr) {
	path := slices.Clone(stack)
	deferred := inDeferredFunc(stack)

	for i, expr := range lhs {
		ident, ok := expr.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}
		obj := pass.TypesInfo.ObjectOf(ident)
		if obj == nil {
			continue
		}

		// A var declaration without values sets the zero value, nil.
		wraps := len(lhs) == len(rhs) && isWrapExpr(rhs[i])
		clean := wraps || len(rhs) == 0 || (len(lhs) == len(rhs) && isNil(pass, rhs[i]))
		facts.writes[obj] = append(facts.writes[obj], assignment{path: path, pos: pos, wraps: wraps, clean: clean})
		if deferred && wraps && resultObjs[obj] {
			facts.deferredWrap[obj] = true
		}
	}
}

// innermostBlock returns the index in path of the innermost block, case
// clause or comm clause, or -1 if there is none.
func innermostBlock(path []ast.Node) int {
	for i, n := range slices.Backward(path) {
		switch n.(type) {
		case *ast.BlockStmt, *ast.CaseClause, *ast.CommClause:
			return i
		}
	}
	return -1
}

// inDeferredFunc reports whether the ancestors on stack include a function
// literal that a defer statement calls: defer func() { ... }().
func inDeferredFunc(stack []ast.Node) bool {
	for i := len(stack) - 1; i >= 2; i-- {
		lit, ok := stack[i].(*ast.FuncLit)
		if !ok {
			continue
		}
		call, ok := stack[i-1].(*ast.CallExpr)
		if !ok || call.Fun != lit {
			continue
		}
		if _, ok := stack[i-2].(*ast.DeferStmt); ok {
			return true
		}
	}
	return false
}

// returnedIdents returns the identifiers a return statement returns: its
// identifier results, or, for a naked return, the named results that an
// enclosing "if err != nil" guarantees to hold an error. A naked return
// elsewhere, such as the last one of a function, returns a nil error.
func returnedIdents(pass *analysis.Pass, site returnSite) []*ast.Ident {
	var idents []*ast.Ident
	if len(site.ret.Results) > 0 {
		for _, result := range site.ret.Results {
			if ident, ok := result.(*ast.Ident); ok {
				idents = append(idents, ident)
			}
		}
		return idents
	}

	if site.results == nil {
		return nil
	}
	for _, field := range site.results.List {
		for _, name := range field.Names {
			if site.nonNil[pass.TypesInfo.ObjectOf(name)] {
				idents = append(idents, name)
			}
		}
	}
	return idents
}

// nonNilGuards returns the variables v for which an ancestor on stack is an
// "if v != nil" (or "if nil != v") whose body holds the current node.
func nonNilGuards(pass *analysis.Pass, stack []ast.Node) map[types.Object]bool {
	guards := make(map[types.Object]bool)
	for i := 0; i+1 < len(stack); i++ {
		ifStmt, ok := stack[i].(*ast.IfStmt)
		if !ok || stack[i+1] != ifStmt.Body {
			continue
		}
		cond, ok := ast.Unparen(ifStmt.Cond).(*ast.BinaryExpr)
		if !ok || cond.Op != token.NEQ {
			continue
		}
		operand := cond.X
		if isNil(pass, operand) {
			operand = cond.Y
		} else if !isNil(pass, cond.Y) {
			continue
		}
		if ident, ok := ast.Unparen(operand).(*ast.Ident); ok {
			if obj := pass.TypesInfo.ObjectOf(ident); obj != nil {
				guards[obj] = true
			}
		}
	}
	return guards
}

// returnsHumaneError checks if any return type is humane.Error
func returnsHumaneError(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil {
		return false
	}

	for _, result := range fn.Type.Results.List {
		sel, ok := result.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "humane" && sel.Sel.Name == "Error" {
			return true
		}
	}

	return false
}

// trackErrorNames records the assigned identifiers that look like error
// variables: err, or a name ending in Err or Error.
func trackErrorNames(lhs []ast.Expr, assigned map[string]bool) {
	for _, expr := range lhs {
		ident, ok := expr.(*ast.Ident)
		if !ok {
			continue
		}

		// Common error variable names
		if ident.Name == errVarName || strings.HasSuffix(ident.Name, "Err") || strings.HasSuffix(ident.Name, "Error") {
			assigned[ident.Name] = true
		}
	}
}

// isWrapExpr reports whether expr is a call that wraps or creates an error
// with context.
func isWrapExpr(expr ast.Expr) bool {
	call, ok := ast.Unparen(expr).(*ast.CallExpr)
	return ok && isErrorWrap(call)
}

func isErrorWrap(call *ast.CallExpr) bool {
	// Check for common wrapping patterns
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		return isWrapSelectorCall(call, fn)

	case *ast.Ident:
		// errors.New is not wrapping but creating new error
		return fn.Name == "Errorf"
	}

	return false
}

// isWrapSelectorCall reports whether a pkg.Func(...) call wraps or creates an
// error with context.
func isWrapSelectorCall(call *ast.CallExpr, fn *ast.SelectorExpr) bool {
	switch fn.Sel.Name {
	case "Errorf":
		// fmt.Errorf with %w
		return formatHasWrapVerb(call)

	case "Wrap", "Wrapf", "WithMessage":
		// humane.Wrap, errors.Wrap, pkg/errors.Wrap
		return true

	case "New":
		// humane.New creates a new error (not wrap, but acceptable)
		ident, ok := fn.X.(*ast.Ident)
		return ok && ident.Name == "humane"
	}

	return false
}

// formatHasWrapVerb reports whether the call's format string contains %w.
func formatHasWrapVerb(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	return ok && strings.Contains(lit.Value, "%w")
}

// checkReturnedIdent reports ident, returned at site, if it is an error
// variable that isn't wrapped at that return.
func checkReturnedIdent(pass *analysis.Pass, reporter *nolint.Reporter, fn *ast.FuncDecl, site returnSite, ident *ast.Ident, facts *bodyFacts) {
	// Check if this is an error variable, or a common error name not
	// explicitly tracked
	if !facts.assigned[ident.Name] && ident.Name != errVarName && !strings.HasSuffix(ident.Name, "Err") {
		return
	}

	obj := pass.TypesInfo.ObjectOf(ident)
	if obj == nil || facts.deferredWrap[obj] || wrappedAt(facts.writes[obj], site) {
		return
	}

	// This is a bare error return
	// Only report if the function has meaningful operations (not just wrapping another call)
	if hasMultipleOperations(fn) {
		reporter.Reportf(site.ret.Pos(),
			"returning error %q without wrapping; add context with humane.Wrap(%s, message, advice...)",
			ident.Name, ident.Name)
	}
}

// wrappedAt reports whether the variable written by writes holds a wrapped
// error, or nil, at the return. The write that decides is the last one
// before the return in a block that also holds the return, so it happens on
// every path to it. A later write before the return, in a nested block that
// isn't another branch of an if or switch holding the return, may or may not
// happen. If one of those conditional writes wraps the variable or sets it to
// nil, as in "if err != nil { err = fmt.Errorf(...) }", the variable counts as
// wrapped; if one sets it to anything else, it doesn't. Without conditional
// writes the deciding write decides. A variable with no deciding write is a
// parameter, which holds an unwrapped error.
func wrappedAt(writes []assignment, site returnSite) bool {
	ret := site.ret
	var last *assignment
	for i := range writes {
		w := &writes[i]
		if w.pos >= ret.Pos() {
			continue
		}
		if b := innermostBlock(w.path); b >= 0 && encloses(w.path[b], ret) {
			last = w
		}
	}

	from := token.NoPos
	if last != nil {
		from = last.pos
	}

	conditionalBare := false
	for _, w := range writes {
		if w.pos <= from || w.pos >= ret.Pos() || otherBranch(w.path, site.path) {
			continue
		}
		if w.clean {
			return true
		}
		conditionalBare = true
	}
	return !conditionalBare && last != nil && last.clean
}

// otherBranch reports whether two nodes, given by their ancestor paths, sit in
// different branches of the same if statement or in different clauses of the
// same switch or select statement, so that one never runs before the other.
func otherBranch(a, b []ast.Node) bool {
	k := 0
	for k < len(a) && k < len(b) && a[k] == b[k] {
		k++
	}
	if k == 0 || k == len(a) || k == len(b) {
		return false
	}

	switch common := a[k-1].(type) {
	case *ast.IfStmt:
		return isBranch(common, a[k]) && isBranch(common, b[k])

	case *ast.BlockStmt:
		if k < 2 {
			return false
		}
		switch a[k-2].(type) {
		case *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
			return true
		}
	}
	return false
}

// isBranch reports whether n is the body or the else branch of ifStmt.
func isBranch(ifStmt *ast.IfStmt, n ast.Node) bool {
	return n == ifStmt.Body || (ifStmt.Else != nil && n == ifStmt.Else)
}

// encloses reports whether block contains n.
func encloses(block, n ast.Node) bool {
	return block.Pos() <= n.Pos() && n.End() <= block.End()
}

func hasMultipleOperations(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}

	// Count meaningful statements (excluding just returns and error checks)
	meaningful := 0
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			meaningful++
		case *ast.AssignStmt:
			// Assignments that aren't just error checks
			if len(node.Lhs) > 1 || (len(node.Lhs) == 1 && !isErrorIdent(node.Lhs[0])) {
				meaningful++
			}
		}
		return meaningful < 3 // Stop early if we've found enough
	})

	return meaningful >= 2
}

func isErrorIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == errVarName || strings.HasSuffix(ident.Name, "Err")
}

// isNil reports whether expr is the predeclared nil.
func isNil(pass *analysis.Pass, expr ast.Expr) bool {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	if !ok {
		return false
	}
	_, isNil := pass.TypesInfo.ObjectOf(ident).(*types.Nil)
	return isNil
}
