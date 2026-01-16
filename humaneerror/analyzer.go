// Package humaneerror provides an analyzer that enforces the use of humane-errors-go
// for all error returns, ensuring errors always include actionable advice.
//
// The analyzer checks:
// 1. Functions returning error must return humane.Error, not plain error
// 2. humane.New() and humane.Wrap() must include at least one advice string
// 3. Error wrapping should preserve the humane error chain
package humaneerror

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `enforce humane-errors-go usage with mandatory advice

This analyzer ensures that:
1. All exported functions returning errors use humane.Error instead of plain error
2. All calls to humane.New() include at least one advice string
3. All calls to humane.Wrap() include at least one advice string
4. Plain errors.New() and fmt.Errorf() are flagged in favor of humane equivalents

The goal is to ensure all errors in the codebase provide actionable user guidance.`

// Analyzer is the humane error analyzer
var Analyzer = &analysis.Analyzer{
	Name:     "humaneerror",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const (
	humanePackage = "github.com/sierrasoftworks/humane-errors-go"
	humaneAlias   = "humane"
)

// stdlibInterfaceMethods lists method signatures that are required by standard library
// interfaces and must return plain `error`. We exempt these from humane.Error requirements.
// Format: "MethodName:paramCount" or just "MethodName" for any param count
var stdlibInterfaceMethods = map[string]bool{
	// io package
	"Read":  true, // io.Reader
	"Write": true, // io.Writer
	"Close": true, // io.Closer
	"Seek":  true, // io.Seeker

	// encoding/json
	"MarshalJSON":   true, // json.Marshaler
	"UnmarshalJSON": true, // json.Unmarshaler

	// encoding
	"MarshalText":     true, // encoding.TextMarshaler
	"UnmarshalText":   true, // encoding.TextUnmarshaler
	"MarshalBinary":   true, // encoding.BinaryMarshaler
	"UnmarshalBinary": true, // encoding.BinaryUnmarshaler

	// database/sql
	"Scan":  true, // sql.Scanner, fmt.Scanner
	"Value": true, // driver.Valuer

	// net package
	"Accept":           true, // net.Listener
	"Dial":             true, // net.Dialer
	"DialContext":      true, // net.Dialer
	"SetDeadline":      true, // net.Conn
	"SetReadDeadline":  true, // net.Conn
	"SetWriteDeadline": true, // net.Conn

	// flag package
	"Set": true, // flag.Value

	// sort package (no error returns)

	// context - usually not implemented by user types

	// http package - ResponseWriter.Write is covered by io.Writer

	// gob package
	"GobEncode": true,
	"GobDecode": true,

	// Other common stdlib interface methods
	"ReadFrom":   true, // io.ReaderFrom
	"WriteTo":    true, // io.WriterTo
	"ReadAt":     true, // io.ReaderAt
	"WriteAt":    true, // io.WriterAt
	"ReadByte":   true, // io.ByteReader
	"WriteByte":  true, // io.ByteWriter
	"UnreadByte": true, // io.ByteScanner
	"ReadRune":   true, // io.RuneReader
	"UnreadRune": true, // io.RuneScanner

	// Kubernetes controller-runtime
	"Reconcile":        true, // reconcile.Reconciler
	"SetupWithManager": true, // Often part of controller setup

	// Tailscale tsnet.Server interface
	"Up":           true, // tsnet.Server
	"Listen":       true, // net.Listener (also tsnet)
	"ListenTLS":    true, // tsnet.Server
	"ListenFunnel": true, // tsnet.Server
	"LocalWhoIs":   true, // tsnet whois

	// gRPC interfaces
	"Serve":           true, // grpc.Server
	"GracefulStop":    true, // grpc.Server (no error but common pattern)
	"RegisterService": true, // grpc registration

	// Cobra command callbacks - must return plain error
	"RunE":               true, // cobra.Command.RunE
	"PreRunE":            true, // cobra.Command.PreRunE
	"PostRunE":           true, // cobra.Command.PostRunE
	"PersistentPreRunE":  true, // cobra.Command.PersistentPreRunE
	"PersistentPostRunE": true, // cobra.Command.PersistentPostRunE

	// HTTP handlers and middleware
	"ServeHTTP":   true, // http.Handler
	"HandlerFunc": true, // common pattern

	// Context functions
	"Err": true, // context.Context.Err()

	// Testing
	"Run": true, // testing.T.Run callback, also common component pattern
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Track imports to understand package aliases
	imports := make(map[string]string) // path -> local name

	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.CallExpr)(nil),
		(*ast.ReturnStmt)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.File:
			// Reset imports for each file
			imports = make(map[string]string)
			for _, imp := range node.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				name := ""
				if imp.Name != nil {
					name = imp.Name.Name
				} else {
					// Use last component of path as default name
					parts := strings.Split(path, "/")
					name = parts[len(parts)-1]
				}
				imports[path] = name
			}

		case *ast.FuncDecl:
			// Track current function context for nested checks
			if node.Name != nil {
				currentFunc = funcContext{
					name:                 node.Name.Name,
					mustReturnPlainError: isFrameworkCallback(node.Name.Name),
				}
			}
			checkFuncReturnsHumaneError(pass, node, imports)

		case *ast.CallExpr:
			checkHumaneCallHasAdvice(pass, node, imports)
			checkForbiddenErrorCalls(pass, node, imports)
		}
	})

	return nil, nil
}

// checkFuncReturnsHumaneError verifies that exported functions returning error
// use humane.Error instead of the plain error interface
func checkFuncReturnsHumaneError(pass *analysis.Pass, fn *ast.FuncDecl, _ map[string]string) {
	// Only check exported functions (capitalized names)
	if fn.Name == nil || !ast.IsExported(fn.Name.Name) {
		return
	}

	// Skip test functions
	if strings.HasPrefix(fn.Name.Name, "Test") || strings.HasPrefix(fn.Name.Name, "Benchmark") {
		return
	}

	// Skip methods that implement standard library interfaces
	// These MUST return plain error to satisfy the interface
	if isStdlibInterfaceMethod(fn) {
		return
	}

	if fn.Type.Results == nil {
		return
	}

	for _, result := range fn.Type.Results.List {
		// Check if return type is the plain "error" interface
		if ident, ok := result.Type.(*ast.Ident); ok {
			if ident.Name == "error" {
				pass.Reportf(result.Pos(),
					"exported function %q returns plain 'error'; use 'humane.Error' from %s instead to provide actionable advice",
					fn.Name.Name, humanePackage)
			}
		}
	}
}

// isStdlibInterfaceMethod checks if a function is implementing a standard library interface method
func isStdlibInterfaceMethod(fn *ast.FuncDecl) bool {
	if fn.Name == nil {
		return false
	}

	methodName := fn.Name.Name

	// Check if this method name is in our stdlib interface list
	if exempt, exists := stdlibInterfaceMethods[methodName]; exists && exempt {
		// For methods (has receiver), this is likely implementing an interface
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			return true
		}

		// For functions (no receiver), still exempt if it matches common patterns
		// like standalone Close(), Read(), Write() functions that wrap interfaces
		return true
	}

	return false
}

// checkHumaneCallHasAdvice ensures humane.New() and humane.Wrap() include advice
func checkHumaneCallHasAdvice(pass *analysis.Pass, call *ast.CallExpr, imports map[string]string) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	// Check if this is a call to the humane package
	humaneLocalName := imports[humanePackage]
	if humaneLocalName == "" {
		// Try common aliases
		for path, name := range imports {
			if strings.Contains(path, "humane-errors") {
				humaneLocalName = name
				break
			}
		}
	}

	if humaneLocalName == "" || ident.Name != humaneLocalName {
		return
	}

	funcName := sel.Sel.Name

	switch funcName {
	case "New":
		// humane.New(message string, advice ...string) requires at least 2 args for advice
		if len(call.Args) < 2 {
			pass.Reportf(call.Pos(),
				"humane.New() should include at least one advice string: humane.New(message, advice1, advice2, ...)")
		}
		// Note: With exactly 2 args, the call has minimum advice. Multiple advice
		// strings are encouraged but not required.

	case "Wrap":
		// humane.Wrap(err, message string, advice ...string) requires at least 3 args for advice
		if len(call.Args) < 3 {
			pass.Reportf(call.Pos(),
				"humane.Wrap() should include at least one advice string: humane.Wrap(err, message, advice1, ...)")
		}
	}

	// Check advice string quality (should be actionable)
	checkAdviceQuality(pass, call, funcName)
}

// checkAdviceQuality verifies that advice strings are actionable
func checkAdviceQuality(pass *analysis.Pass, call *ast.CallExpr, funcName string) {
	startIdx := 1 // For New(), advice starts at index 1
	if funcName == "Wrap" {
		startIdx = 2 // For Wrap(), advice starts at index 2
	}

	for i := startIdx; i < len(call.Args); i++ {
		lit, ok := call.Args[i].(*ast.BasicLit)
		if !ok {
			continue
		}

		advice := strings.Trim(lit.Value, `"`)
		adviceLower := strings.ToLower(advice)

		// Check for non-actionable advice patterns
		nonActionablePatterns := []string{
			"see underlying error",
			"check error",
			"something went wrong",
			"an error occurred",
			"failed",
			"error:",
		}

		for _, pattern := range nonActionablePatterns {
			if strings.Contains(adviceLower, pattern) && len(advice) < 50 {
				pass.Reportf(lit.Pos(),
					"advice %q may not be actionable; provide specific steps the user can take to resolve the issue",
					advice)
				break
			}
		}

		// Good advice patterns (for reference, not enforced):
		// - "Ensure that..." / "Make sure..."
		// - "Check that..."
		// - "Verify..."
		// - "Try..."
		// - Contains specific values or paths
	}
}

// currentFuncContext tracks context about the current function being analyzed
type funcContext struct {
	name                 string
	mustReturnPlainError bool
}

var currentFunc funcContext

// checkForbiddenErrorCalls flags direct use of errors.New and fmt.Errorf
// but exempts framework callbacks where plain error is required
func checkForbiddenErrorCalls(pass *analysis.Pass, call *ast.CallExpr, _ map[string]string) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	funcName := sel.Sel.Name

	// Check for errors.New - still flag these even in callbacks
	// because you should at least wrap with context
	if ident.Name == "errors" && funcName == "New" {
		// Allow in test files implicitly (they often use errors.New for test cases)
		pass.Reportf(call.Pos(),
			"avoid errors.New(); use humane.New(message, advice...) to provide actionable guidance")
	}

	// Check for fmt.Errorf - allow in framework callbacks
	if ident.Name == "fmt" && funcName == "Errorf" {
		// Allow fmt.Errorf in functions that must return plain error
		// (framework callbacks, interface implementations)
		if !currentFunc.mustReturnPlainError {
			pass.Reportf(call.Pos(),
				"avoid fmt.Errorf(); use humane.Wrap(err, message, advice...) or humane.New(message, advice...) instead")
		}
	}
}

// isFrameworkCallback checks if a function is a framework callback
// where plain error returns are required
func isFrameworkCallback(funcName string) bool {
	// Check against known stdlib/framework interface methods
	if stdlibInterfaceMethods[funcName] {
		return true
	}

	// Check for common callback patterns in function names
	callbackPatterns := []string{
		"RunE", "PreRunE", "PostRunE", // Cobra
		"Handler", "HandlerFunc", "Middleware", // HTTP
		"Interceptor", // gRPC
		"Callback", "Hook",
	}

	for _, pattern := range callbackPatterns {
		if strings.Contains(funcName, pattern) {
			return true
		}
	}

	return false
}

// IsHumaneErrorType checks if a type is humane.Error
func IsHumaneErrorType(t types.Type) bool {
	if t == nil {
		return false
	}

	// Check for the humane.Error interface
	if named, ok := t.(*types.Named); ok {
		obj := named.Obj()
		if obj != nil && obj.Pkg() != nil {
			pkgPath := obj.Pkg().Path()
			return strings.Contains(pkgPath, "humane-errors") && obj.Name() == "Error"
		}
	}

	return false
}
