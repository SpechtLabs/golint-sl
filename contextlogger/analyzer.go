// Package contextlogger provides an analyzer that enforces consistent context-based logging patterns.
//
// Inspired by the compute-blade-agent pattern:
//
//	func IntoContext(ctx context.Context, logger *Logger) context.Context
//	func FromContext(ctx context.Context) *Logger
//
// This ensures structured logging with proper context propagation.
package contextlogger

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

// Doc is the contextlogger analyzer's documentation.
const Doc = `enforce context-based logging patterns

This analyzer ensures:
1. Functions with context parameter use log.FromContext(ctx) not global logger
2. Logger is propagated through context, not as a separate parameter
3. Log calls include relevant context fields (request ID, trace ID, etc.)

The context logger pattern provides:
- Automatic trace correlation
- Consistent log enrichment across the call stack
- Cleaner function signatures

Example:
    // Good: Extract logger from context
    func handleRequest(ctx context.Context) {
        logger := log.FromContext(ctx)
        logger.Info("handling request")
    }

    // Bad: Using global logger when context is available
    func handleRequest(ctx context.Context) {
        log.Info("handling request")  // Loses context
    }`

// Analyzer enforces context-based logging patterns.
var Analyzer = &analysis.Analyzer{
	Name:     "contextlogger",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// GlobalLoggerPatterns are patterns that indicate global logger usage
// These should use context-derived loggers instead for proper tracing.
//
// Each pattern is a package name and a function name; a trailing "()" is
// only decoration. A call matches when it calls exactly that function of a
// package with that name (however it was imported), or that method on a
// package-level variable with that name, such as var log = logrus.New().
var GlobalLoggerPatterns = []string{
	"log.Info",
	"log.Infof",
	"log.Error",
	"log.Errorf",
	"log.Warn",
	"log.Warnf",
	"log.Debug",
	"log.Debugf",
	"log.Fatal",
	"log.Fatalf",
	"log.Fatalln",
	"log.Print",
	"log.Printf",
	"log.Println",
	"zap.L()",
	"zap.S()",
	"logrus.Info",
	"logrus.Infof",
	"logrus.Error",
	"logrus.Errorf",
	"logrus.Warn",
	"logrus.Warnf",
	"logrus.Debug",
	"logrus.Debugf",
	"logrus.WithField",
	"logrus.WithFields",
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok {
			return
		}

		// Check if function has context parameter
		if !hasContextParameter(fn) {
			return
		}

		// Skip init and main
		if fn.Name != nil && (fn.Name.Name == "init" || fn.Name.Name == "main") {
			return
		}

		if fn.Body == nil {
			return
		}

		// Check for global logger usage
		checkGlobalLoggerUsage(pass, reporter, fn)

		// Check for logger passed as parameter (should use context instead)
		checkLoggerParameter(reporter, fn)
	})

	return nil, nil
}

// hasContextParameter checks if a function accepts context.Context
func hasContextParameter(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil {
		return false
	}

	for _, param := range fn.Type.Params.List {
		paramType := types.ExprString(param.Type)
		if strings.Contains(paramType, "context.Context") || paramType == "Context" {
			return true
		}
	}

	return false
}

// checkGlobalLoggerUsage detects usage of global logger when context is available
func checkGlobalLoggerUsage(pass *analysis.Pass, reporter *nolint.Reporter, fn *ast.FuncDecl) {
	usesFromContext := false
	var globalLoggerCalls []*ast.CallExpr

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check if FromContext is used
		if isFromContextCall(call) {
			usesFromContext = true
		}

		// Check for global logger patterns
		if isGlobalLoggerCall(pass, call) {
			globalLoggerCalls = append(globalLoggerCalls, call)
		}

		return true
	})

	// If context is available but global logger is used without FromContext
	if !usesFromContext && len(globalLoggerCalls) > 0 {
		for _, call := range globalLoggerCalls {
			reporter.Reportf(call.Pos(),
				"function has context parameter but uses global logger; use log.FromContext(ctx) instead")
		}
	}
}

// checkLoggerParameter detects logger passed as parameter (anti-pattern)
func checkLoggerParameter(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	// Only a logger passed alongside a context is flagged (the logger should
	// come from the context)
	if fn.Type.Params == nil || !hasContextParameter(fn) {
		return
	}

	// Check for common logger types passed as parameter
	loggerPatterns := []string{
		"*zap.Logger",
		"*zap.SugaredLogger",
		"*logrus.Logger",
		"*logrus.Entry",
		"*slog.Logger",
		"*otelzap.Logger",
	}

	for _, param := range fn.Type.Params.List {
		paramType := types.ExprString(param.Type)
		for _, pattern := range loggerPatterns {
			if strings.Contains(paramType, pattern) {
				reporter.Reportf(param.Pos(),
					"logger passed as parameter alongside context; consider using log.FromContext(ctx) pattern instead")
			}
		}
	}
}

// ContextLoggerInfo contains information about logger patterns in a package
type ContextLoggerInfo struct {
	HasFromContext     bool
	HasIntoContext     bool
	GlobalLoggerCalls  int
	ContextLoggerCalls int
}

// AnalyzeContextLogger returns information about context logger pattern usage
func AnalyzeContextLogger(pass *analysis.Pass) *ContextLoggerInfo {
	info := &ContextLoggerInfo{}
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if node.Name != nil {
				if node.Name.Name == "FromContext" {
					info.HasFromContext = true
				}
				if node.Name.Name == "IntoContext" {
					info.HasIntoContext = true
				}
			}

		case *ast.CallExpr:
			if isFromContextCall(node) {
				info.ContextLoggerCalls++
			}
			if isGlobalLoggerCall(pass, node) {
				info.GlobalLoggerCalls++
			}
		}
	})

	return info
}

// isFromContextCall reports whether call calls a function or method whose own
// name mentions FromContext, such as log.FromContext(ctx). A call chained on
// its result, such as log.FromContext(ctx).Info("x"), is not one.
func isFromContextCall(call *ast.CallExpr) bool {
	var name string
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		name = fun.Name
	case *ast.SelectorExpr:
		name = fun.Sel.Name
	}
	return strings.Contains(name, "FromContext")
}

// isGlobalLoggerCall reports whether call matches one of GlobalLoggerPatterns.
// The callee is resolved with go/types: a package-level function is
// qualified by its package's name, and a method by the package-level variable
// it is called on. Anything else, such as a method on a local variable or on
// the result of another call, is not a global logger call.
func isGlobalLoggerCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if !ok || fn.Pkg() == nil {
		return false
	}

	qualifier := fn.Pkg().Name()
	if sig, ok := fn.Type().(*types.Signature); ok && sig.Recv() != nil {
		qualifier = packageVarName(pass, call)
		if qualifier == "" {
			return false
		}
	}

	for _, pattern := range GlobalLoggerPatterns {
		pkg, name, ok := strings.Cut(strings.TrimSuffix(pattern, "()"), ".")
		if ok && pkg == qualifier && name == fn.Name() {
			return true
		}
	}
	return false
}

// packageVarName returns the name of the package-level variable a method call
// is made on (log in log.Info("x") after var log = logrus.New()), or "".
func packageVarName(pass *analysis.Pass, call *ast.CallExpr) string {
	sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	ident, ok := ast.Unparen(sel.X).(*ast.Ident)
	if !ok {
		return ""
	}
	v, ok := pass.TypesInfo.Uses[ident].(*types.Var)
	if !ok || v.IsField() || v.Pkg() == nil || v.Parent() != v.Pkg().Scope() {
		return ""
	}
	return v.Name()
}
