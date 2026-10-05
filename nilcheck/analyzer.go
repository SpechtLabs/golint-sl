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

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the nilcheck analyzer's documentation.
const Doc = `enforce nil checks on pointer parameters before use

This analyzer detects:
1. Pointer parameters used without nil check
2. Pointer fields accessed without nil check
3. Interface values used without nil check

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
	"*http.Request":       true,
	"http.ResponseWriter": true,

	// Context (interface, but trusted)
	"context.Context": true,

	// Kubernetes controller-runtime
	"*reconcile.Request": true,
	"reconcile.Request":  true,

	// Common loggers - never nil in practice
	"*zap.Logger":        true,
	"*zap.SugaredLogger": true,
	"*log.Logger":        true,

	// gRPC
	"*grpc.Server":     true,
	"*grpc.ClientConn": true,

	// Protobuf types are validated by framework
	"proto.Message": true,

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
	// Generated proto types
	".pb.go",
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

	// Track which parameters have been nil-checked
	checkedParams := make(map[string]bool)

	// First pass: find nil checks
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			// Check for: if x == nil or if x != nil
			checkedParam := extractNilCheck(ifStmt.Cond)
			if checkedParam != "" {
				checkedParams[checkedParam] = true
			}
		}
		return true
	})

	// Second pass: find usages of unchecked pointers
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		// Skip the nil check conditions themselves
		if ifStmt, ok := n.(*ast.IfStmt); ok {
			// Skip checking inside the nil-check's then block if it's an early return
			if checkedParam := extractNilCheck(ifStmt.Cond); checkedParam != "" && isEarlyReturnBlock(ifStmt.Body) {
				// After this if block, the param is effectively checked
				checkedParams[checkedParam] = true
			}
		}

		// Check for pointer dereference
		switch node := n.(type) {
		case *ast.SelectorExpr:
			// x.Field - check if x is an unchecked pointer param
			if ident, ok := node.X.(*ast.Ident); ok && ptrParams[ident.Name] && !checkedParams[ident.Name] {
				reporter.Reportf(node.Pos(),
					"pointer parameter %q used without nil check; add 'if %s == nil { return ... }' at function start",
					ident.Name, ident.Name)
				// Mark as reported to avoid duplicate reports
				checkedParams[ident.Name] = true
			}

		case *ast.StarExpr:
			// *x - explicit dereference
			if ident, ok := node.X.(*ast.Ident); ok && ptrParams[ident.Name] && !checkedParams[ident.Name] {
				reporter.Reportf(node.Pos(),
					"pointer parameter %q dereferenced without nil check; add 'if %s == nil { return ... }' at function start",
					ident.Name, ident.Name)
				checkedParams[ident.Name] = true
			}

		case *ast.IndexExpr:
			// x[i] - could be slice/map from pointer
			if ident, ok := node.X.(*ast.Ident); ok && ptrParams[ident.Name] && !checkedParams[ident.Name] {
				reporter.Reportf(node.Pos(),
					"pointer parameter %q indexed without nil check",
					ident.Name)
				checkedParams[ident.Name] = true
			}
		}

		return true
	})
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

// collectPointerParams returns a map of parameter names that are pointers
func collectPointerParams(pass *analysis.Pass, fn *ast.FuncDecl) map[string]bool {
	params := make(map[string]bool)

	if fn.Type.Params == nil {
		return params
	}

	for _, field := range fn.Type.Params.List {
		// Skip trusted pointer types (framework types that are never nil),
		// and parameters whose type is not a pointer
		if isTrustedType(types.ExprString(field.Type)) || !isPointerParam(pass, field) {
			continue
		}

		for _, name := range field.Names {
			// Skip trusted parameter names
			if trustedParamNames[name.Name] {
				continue
			}
			params[name.Name] = true
		}
	}

	return params
}

// isPointerParam reports whether the parameter's type is a pointer (or a
// nilable interface) that is not trusted to be non-nil.
func isPointerParam(pass *analysis.Pass, field *ast.Field) bool {
	switch t := field.Type.(type) {
	case *ast.StarExpr:
		// *T - pointer type
		// Check if it's a trusted type
		return !isTrustedType("*" + types.ExprString(t.X))
	case *ast.Ident:
		// Could be an interface or type alias
		// Check with type info if available
		return isPointerIdent(pass, t)
	case *ast.InterfaceType:
		// interface{} can be nil - but often used with type assertions
		// Skip for now as it causes many false positives
		return false
	}

	// pkg.Type and everything else is not treated as a pointer
	return false
}

// isPointerIdent reports whether a named parameter type is a pointer or a
// nilable interface other than error and Context.
func isPointerIdent(pass *analysis.Pass, t *ast.Ident) bool {
	obj := pass.TypesInfo.ObjectOf(t)
	if obj == nil {
		return false
	}

	if _, ok := obj.Type().Underlying().(*types.Pointer); ok {
		return true
	}

	// Also check for interfaces (can be nil)
	// But skip common trusted interfaces
	if _, ok := obj.Type().Underlying().(*types.Interface); ok {
		// Skip error interface and context
		return t.Name != "error" && t.Name != "Context"
	}

	return false
}

// extractNilCheck checks if a condition is a nil check and returns the variable name
func extractNilCheck(cond ast.Expr) string {
	binExpr, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return ""
	}

	// Check for x == nil or x != nil
	if binExpr.Op != token.EQL && binExpr.Op != token.NEQ {
		return ""
	}

	var varName string

	// Check X == nil or X != nil
	if ident, ok := binExpr.X.(*ast.Ident); ok && isNilIdent(binExpr.Y) {
		varName = ident.Name
	}

	// Check nil == X or nil != X
	if ident, ok := binExpr.Y.(*ast.Ident); ok && isNilIdent(binExpr.X) {
		varName = ident.Name
	}

	return varName
}

func isNilIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "nil"
}

// isEarlyReturnBlock checks if a block ends with a return statement
func isEarlyReturnBlock(block *ast.BlockStmt) bool {
	if len(block.List) == 0 {
		return false
	}

	lastStmt := block.List[len(block.List)-1]
	_, isReturn := lastStmt.(*ast.ReturnStmt)
	return isReturn
}
