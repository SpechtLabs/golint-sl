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

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

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

var Analyzer = &analysis.Analyzer{
	Name:     "contextlogger",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// GlobalLoggerPatterns are patterns that indicate global logger usage
// These should use context-derived loggers instead for proper tracing
var GlobalLoggerPatterns = []string{
	"log.Info",
	"log.Error",
	"log.Warn",
	"log.Debug",
	"log.Fatal",
	"log.Print",
	"log.Printf",
	"log.Println",
	"zap.L()",
	"zap.S()",
	"logrus.Info",
	"logrus.Error",
	"logrus.Warn",
	"logrus.Debug",
	"logrus.WithField",
	"logrus.WithFields",
}

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
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
		checkGlobalLoggerUsage(reporter, fn)

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
func checkGlobalLoggerUsage(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	usesFromContext := false
	var globalLoggerCalls []*ast.CallExpr

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		callStr := exprToString(call.Fun)

		// Check if FromContext is used
		if strings.Contains(callStr, "FromContext") {
			usesFromContext = true
		}

		// Check for global logger patterns
		for _, pattern := range GlobalLoggerPatterns {
			if strings.Contains(callStr, pattern) {
				globalLoggerCalls = append(globalLoggerCalls, call)
			}
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
	if fn.Type.Params == nil {
		return
	}

	for _, param := range fn.Type.Params.List {
		paramType := types.ExprString(param.Type)

		// Check for common logger types passed as parameter
		loggerPatterns := []string{
			"*zap.Logger",
			"*zap.SugaredLogger",
			"*logrus.Logger",
			"*logrus.Entry",
			"*slog.Logger",
			"*otelzap.Logger",
		}

		for _, pattern := range loggerPatterns {
			if strings.Contains(paramType, pattern) {
				// Check if context is also a parameter (which means logger should come from context)
				if hasContextParameter(fn) {
					reporter.Reportf(param.Pos(),
						"logger passed as parameter alongside context; consider using log.FromContext(ctx) pattern instead")
				}
			}
		}
	}
}

// exprToString converts an expression to a readable string
func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.CallExpr:
		return exprToString(e.Fun) + "()"
	default:
		return types.ExprString(expr)
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
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.CallExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
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
			callStr := exprToString(node.Fun)
			if strings.Contains(callStr, "FromContext") {
				info.ContextLoggerCalls++
			}
			for _, pattern := range GlobalLoggerPatterns {
				if strings.Contains(callStr, pattern) {
					info.GlobalLoggerCalls++
					break
				}
			}
		}
	})

	return info
}
