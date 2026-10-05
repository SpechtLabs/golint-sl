// Package nilcheck provides an analyzer that enforces nil checks on pointer parameters.
//
// Nil pointer dereferences cause panics at runtime. This analyzer ensures that
// pointer parameters are checked for nil before being used.
package nilcheck

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the nilcheck analyzer's documentation.
const Doc = `enforce nil checks on pointer parameters before use

This analyzer reports a pointer parameter that is dereferenced (a field
access or method call, *p, or p[i]) where it isn't known to be non-nil.
A use is known to be non-nil when it comes:
- inside an if whose condition rules nil out (if p != nil { ... },
  if p == nil { ... } else { ... }, compound conditions with && and ||)
- after p != nil && in the same condition, or after p == nil ||
- after an if whose condition holds whenever p is nil and whose body
  returns, panics, exits, breaks out, or assigns p

Interface and type-parameter parameters are not pointers and are not
checked. Framework types and parameter names that are never nil in
practice (*testing.T, *http.Request, ctx, cfg, ...) are skipped, as are
generated and mock files.

Every pointer parameter should be validated at the start of a function:

    func ProcessUser(user *User) error {
        if user == nil {
            return errors.New("user cannot be nil")
        }
        // Now safe to use user
        fmt.Println(user.Name)
    }

This prevents nil pointer panics and provides better error messages.`

// Analyzer reports pointer parameters used without a nil check.
var Analyzer = &analysis.Analyzer{
	Name:     "nilcheck",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Types that are guaranteed non-nil by their callers (framework types)
var trustedPointerTypes = map[string]bool{
	// Testing
	"*testing.T": true,
	"*testing.B": true,
	"*testing.M": true,
	"*testing.F": true,

	// Gin framework
	"*gin.Context":     true,
	"*gin.Engine":      true,
	"*gin.RouterGroup": true,

	// Cobra CLI
	"*cobra.Command": true,

	// HTTP
	"*http.Request": true,

	// Kubernetes controller-runtime
	"*reconcile.Request": true,

	// Common loggers - never nil in practice
	"*zap.Logger":        true,
	"*zap.SugaredLogger": true,
	"*log.Logger":        true,

	// gRPC
	"*grpc.Server":     true,
	"*grpc.ClientConn": true,

	// HTTP response - typically guaranteed non-nil when error is nil
	"*http.Response": true,

	// OS exec - typically from Command() which never returns nil
	"*exec.Cmd": true,

	// Kubernetes API types
	"*api.Config":  true, // clientcmd kubeconfig
	"*rest.Config": true, // client-go rest config

	// Flag/pflag sets - from Flags() which never returns nil
	"*flag.FlagSet":  true,
	"*pflag.FlagSet": true,

	// Viper - from Get() which typically doesn't return nil
	"*viper.Viper": true,
}

// trustedTypePatterns are partial matches for type names that indicate trusted types
var trustedTypePatterns = []string{
	// Kubernetes CRD types often have guaranteed non-nil from reconciler
	"v1alpha1.",
	"v1beta1.",
	"v1.",
}

// trustedParamNames are parameter names that are typically framework-provided
var trustedParamNames = map[string]bool{
	"req":    true, // http.Request
	"res":    true, // response
	"resp":   true, // http.Response - typically from Do() which guarantees non-nil when err==nil
	"w":      true, // http.ResponseWriter
	"r":      true, // http.Request
	"ctx":    true, // context.Context
	"c":      true, // gin.Context
	"t":      true, // testing.T
	"b":      true, // testing.B
	"ts":     true, // test server
	"logger": true, // *zap.Logger
	"log":    true, // logger
	"l":      true, // logger
	"cmd":    true, // exec.Cmd or cobra.Command - typically from Command() which guarantees non-nil
	"f":      true, // flagset - typically from command.Flags() which guarantees non-nil
	"opts":   true, // options - typically has a default or is validated
	"opt":    true, // option
	"cfg":    true, // config - typically validated
	"config": true, // config
}

// File patterns to skip (generated code, etc.)
var skipFilePatterns = []string{
	"zz_generated",
	".pb.go",
	"_gen.go",
	"mock_",
	"mocks/",
}

// terminatingCalls are the functions and methods, by name, that never return
// to the caller: os.Exit, log.Fatal, t.Fatal, t.Skip and their variants.
var terminatingCalls = map[string]bool{
	"Exit": true, "Goexit": true,
	"Fatal": true, "Fatalf": true, "Fatalln": true, "Fatalw": true,
	"Panic": true, "Panicf": true, "Panicln": true, "Panicw": true,
	"FailNow": true, "Skip": true, "Skipf": true, "SkipNow": true,
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

		// Skip generated files
		filename := pass.Fset.Position(fn.Pos()).Filename
		for _, pattern := range skipFilePatterns {
			if strings.Contains(filename, pattern) {
				return
			}
		}

		checkFunction(reporter, pass, fn)
	})

	return nil, nil
}

func checkFunction(reporter *nolint.Reporter, pass *analysis.Pass, fn *ast.FuncDecl) {
	// Collect pointer parameters
	ptrParams := collectPointerParams(pass, fn)
	if len(ptrParams) == 0 {
		return
	}

	// The ends of the ifs after which each parameter is known to be non-nil
	checkedAfter := collectTerminatingChecks(pass, fn.Body, ptrParams)

	// Report every parameter once, at its first unguarded use
	reported := make(map[*types.Var]bool)

	var stack []ast.Node
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		stack = append(stack, n)

		param, verb := dereferencedParam(pass, n, ptrParams)
		if param == nil || reported[param] || isGuarded(pass, param, stack, checkedAfter[param]) {
			return true
		}
		reported[param] = true

		switch verb {
		case "indexed":
			reporter.Reportf(n.Pos(), "pointer parameter %q indexed without nil check", param.Name())
		default:
			reporter.Reportf(n.Pos(),
				"pointer parameter %q %s without nil check; add 'if %s == nil { return ... }' at function start",
				param.Name(), verb, param.Name())
		}
		return true
	})
}

// dereferencedParam returns the pointer parameter n dereferences, and how:
// p.Field or p.Method() ("used"), *p ("dereferenced") or p[i] ("indexed").
func dereferencedParam(pass *analysis.Pass, n ast.Node, ptrParams map[*types.Var]bool) (*types.Var, string) {
	var x ast.Expr
	var verb string

	switch node := n.(type) {
	case *ast.SelectorExpr:
		x, verb = node.X, "used"
	case *ast.StarExpr:
		x, verb = node.X, "dereferenced"
	case *ast.IndexExpr:
		x, verb = node.X, "indexed"
	default:
		return nil, ""
	}

	param := paramOf(pass, x, ptrParams)
	if param == nil {
		return nil, ""
	}
	return param, verb
}

// paramOf returns the pointer parameter expr refers to, or nil.
func paramOf(pass *analysis.Pass, expr ast.Expr, ptrParams map[*types.Var]bool) *types.Var {
	ident, ok := ast.Unparen(expr).(*ast.Ident)
	if !ok {
		return nil
	}
	v, ok := pass.TypesInfo.Uses[ident].(*types.Var)
	if !ok || !ptrParams[v] {
		return nil
	}
	return v
}

// isGuarded reports whether the innermost node of stack, a use of param,
// can only run when param is non-nil.
func isGuarded(pass *analysis.Pass, param *types.Var, stack []ast.Node, checkedAfter []token.Pos) bool {
	use := stack[len(stack)-1]
	for _, end := range checkedAfter {
		if end <= use.Pos() {
			return true
		}
	}

	for i := len(stack) - 2; i >= 0; i-- {
		if guardsChild(pass, param, stack[i], stack[i+1]) {
			return true
		}
	}
	return false
}

// guardsChild reports whether parent only runs child when param is non-nil:
// the body of an if whose condition rules nil out, the else of one whose
// condition holds for nil, and the right operand of p != nil && or p == nil ||.
func guardsChild(pass *analysis.Pass, param *types.Var, parent, child ast.Node) bool {
	switch p := parent.(type) {
	case *ast.IfStmt:
		if child == ast.Node(p.Body) {
			return impliesNonNil(pass, p.Cond, param, true)
		}
		if p.Else != nil && child == ast.Node(p.Else) {
			return impliesNonNil(pass, p.Cond, param, false)
		}

	case *ast.BinaryExpr:
		if child != ast.Node(p.Y) {
			return false
		}
		switch p.Op {
		case token.LAND:
			return impliesNonNil(pass, p.X, param, true)
		case token.LOR:
			return impliesNonNil(pass, p.X, param, false)
		}
	}
	return false
}

// impliesNonNil reports whether cond evaluating to outcome means param is
// not nil: p != nil being true, p == nil being false, and the && and ||
// combinations and negations of those.
func impliesNonNil(pass *analysis.Pass, cond ast.Expr, param *types.Var, outcome bool) bool {
	switch c := ast.Unparen(cond).(type) {
	case *ast.UnaryExpr:
		return c.Op == token.NOT && impliesNonNil(pass, c.X, param, !outcome)

	case *ast.BinaryExpr:
		switch c.Op {
		case token.LAND:
			// Both operands are true when a && b is
			return outcome && (impliesNonNil(pass, c.X, param, true) || impliesNonNil(pass, c.Y, param, true))
		case token.LOR:
			// Both operands are false when a || b is
			return !outcome && (impliesNonNil(pass, c.X, param, false) || impliesNonNil(pass, c.Y, param, false))
		case token.NEQ:
			return outcome && isNilComparison(pass, c, param)
		case token.EQL:
			return !outcome && isNilComparison(pass, c, param)
		}
	}
	return false
}

// isNilComparison reports whether cmp compares param with nil, either way round.
func isNilComparison(pass *analysis.Pass, cmp *ast.BinaryExpr, param *types.Var) bool {
	isNil := func(e ast.Expr) bool { return pass.TypesInfo.Types[e].IsNil() }
	ptrParams := map[*types.Var]bool{param: true}
	return (paramOf(pass, cmp.X, ptrParams) != nil && isNil(cmp.Y)) ||
		(paramOf(pass, cmp.Y, ptrParams) != nil && isNil(cmp.X))
}

// collectTerminatingChecks returns, per parameter, the end positions of the
// ifs that leave it non-nil afterwards: their condition holds whenever the
// parameter is nil, and their body leaves the function or loop, or assigns
// the parameter.
func collectTerminatingChecks(pass *analysis.Pass, body *ast.BlockStmt, ptrParams map[*types.Var]bool) map[*types.Var][]token.Pos {
	checkedAfter := make(map[*types.Var][]token.Pos)

	ast.Inspect(body, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		for param := range ptrParams {
			if impliesNonNil(pass, ifStmt.Cond, param, false) &&
				(isTerminatingBlock(pass, ifStmt.Body) || assigns(pass, ifStmt.Body, param)) {
				checkedAfter[param] = append(checkedAfter[param], ifStmt.End())
			}
		}
		return true
	})

	return checkedAfter
}

// isTerminatingBlock checks if a block ends with a statement that doesn't
// fall through: return, break, continue, goto, panic, or a call that exits.
func isTerminatingBlock(pass *analysis.Pass, block *ast.BlockStmt) bool {
	if len(block.List) == 0 {
		return false
	}

	switch last := block.List[len(block.List)-1].(type) {
	case *ast.ReturnStmt, *ast.BranchStmt:
		return true
	case *ast.ExprStmt:
		call, ok := last.X.(*ast.CallExpr)
		if !ok {
			return false
		}
		switch callee := typeutil.Callee(pass.TypesInfo, call).(type) {
		case *types.Builtin:
			return callee.Name() == "panic"
		case *types.Func:
			return terminatingCalls[callee.Name()]
		}
	}
	return false
}

// assigns reports whether block assigns param anywhere.
func assigns(pass *analysis.Pass, block *ast.BlockStmt, param *types.Var) bool {
	ptrParams := map[*types.Var]bool{param: true}
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if assign, ok := n.(*ast.AssignStmt); ok {
			for _, lhs := range assign.Lhs {
				found = found || paramOf(pass, lhs, ptrParams) != nil
			}
		}
		return !found
	})
	return found
}

// isTrustedType checks if a type string matches any trusted type patterns
func isTrustedType(typeStr string) bool {
	// Check exact matches
	if trustedPointerTypes[typeStr] {
		return true
	}

	// Check patterns
	for _, pattern := range trustedTypePatterns {
		if strings.Contains(typeStr, pattern) {
			return true
		}
	}

	return false
}

// collectPointerParams returns the parameters whose type is a pointer
func collectPointerParams(pass *analysis.Pass, fn *ast.FuncDecl) map[*types.Var]bool {
	params := make(map[*types.Var]bool)

	for _, field := range fn.Type.Params.List {
		// Skip trusted pointer types (framework types that are never nil),
		// and parameters whose type is not a pointer
		if isTrustedType(types.ExprString(field.Type)) || !isPointer(pass.TypesInfo.TypeOf(field.Type)) {
			continue
		}

		for _, name := range field.Names {
			// Skip trusted parameter names
			if trustedParamNames[name.Name] {
				continue
			}
			if v, ok := pass.TypesInfo.Defs[name].(*types.Var); ok {
				params[v] = true
			}
		}
	}

	return params
}

// isPointer reports whether t is a pointer type, named or not. Interfaces
// and type parameters are not pointers, even when they hold one.
func isPointer(t types.Type) bool {
	if t == nil {
		return false
	}
	if _, ok := types.Unalias(t).(*types.TypeParam); ok {
		return false
	}
	_, ok := t.Underlying().(*types.Pointer)
	return ok
}
