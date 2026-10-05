// Package nopanic provides an analyzer that ensures library code never panics.
//
// Library functions should return errors instead of panicking. Panics should only
// be used in main packages or for truly unrecoverable programmer errors.
package nopanic

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

// Doc is the nopanic analyzer's documentation.
const Doc = `ensure library code returns errors instead of panicking

This analyzer detects, in non-main packages outside _test.go files and
outside init and TestMain:
1. panic() calls
2. Fatal and Panic log calls: the log and logrus package functions, and the
   Fatal* and Panic* methods of the log, logrus and zap loggers, however the
   logger is reached (a variable, a struct field, zap.L(), logrus.WithError)

Calling a Must* helper such as regexp.MustCompile is not reported; those
are the idiomatic way to build package-level values. A Must* function of
your own that calls panic is reported like any other panic.

Library code should return errors and let the caller decide how to handle them.
Panics make code difficult to use as a library and can crash the entire program.

Good pattern:
    func ParseConfig(data []byte) (*Config, error) {
        var cfg Config
        if err := json.Unmarshal(data, &cfg); err != nil {
            return nil, fmt.Errorf("invalid config: %w", err)
        }
        return &cfg, nil
    }

Bad pattern:
    func MustParseConfig(data []byte) *Config {
        var cfg Config
        if err := json.Unmarshal(data, &cfg); err != nil {
            panic(err)  // Crashes the program!
        }
        return &cfg
    }`

// Analyzer reports panics in library code that should return errors.
var Analyzer = &analysis.Analyzer{
	Name:     "nopanic",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Functions where panic is acceptable (initialization, tests)
var allowedPanicFunctions = map[string]bool{
	"init":     true,
	"TestMain": true,
}

// loggerPackages are the import paths of the logging packages whose Fatal and
// Panic functions and methods end the program.
var loggerPackages = map[string]bool{
	"log":                        true,
	"github.com/sirupsen/logrus": true,
	"go.uber.org/zap":            true,
}

// terminatingLogFuncs are the names of the logging functions and methods
// that exit or panic after logging.
var terminatingLogFuncs = map[string]bool{
	"Fatal": true, "Fatalf": true, "Fatalln": true, "Fatalw": true,
	"Panic": true, "Panicf": true, "Panicln": true, "Panicw": true,
}

func run(pass *analysis.Pass) (any, error) {
	// Skip main packages
	if pass.Pkg.Name() == "main" {
		return nil, nil
	}

	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// The stack gives each call its file and its enclosing function
	// declaration, so package-level code is never attributed to the function
	// declared before it.
	insp.WithStack([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}

		file := stack[0].(*ast.File)
		if strings.HasSuffix(pass.Fset.Position(file.Pos()).Filename, "_test.go") {
			return true
		}

		// Skip allowed functions
		if fn := enclosingFuncDecl(stack); fn != nil && fn.Recv == nil && allowedPanicFunctions[fn.Name.Name] {
			return true
		}

		checkPanicCall(pass, reporter, n.(*ast.CallExpr))
		return true
	})

	return nil, nil
}

// enclosingFuncDecl returns the function declaration the innermost node of
// stack sits in, or nil for package-level code.
func enclosingFuncDecl(stack []ast.Node) *ast.FuncDecl {
	for _, n := range stack {
		if fn, ok := n.(*ast.FuncDecl); ok {
			return fn
		}
	}
	return nil
}

func checkPanicCall(pass *analysis.Pass, reporter *nolint.Reporter, call *ast.CallExpr) {
	switch callee := typeutil.Callee(pass.TypesInfo, call).(type) {
	case *types.Builtin:
		// Check for direct panic calls
		if callee.Name() == "panic" {
			reporter.Reportf(call.Pos(),
				"panic() in library code; return an error instead to let callers handle failures gracefully")
		}

	case *types.Func:
		// Note: Must* functions that panic are generally acceptable
		// as they follow Go conventions (e.g., regexp.MustCompile)
		if callee.Pkg() == nil || !loggerPackages[callee.Pkg().Path()] || !terminatingLogFuncs[callee.Name()] {
			return
		}

		// Package-level log.Fatal, logrus.Panicf and friends
		if callee.Signature().Recv() == nil {
			reporter.Reportf(call.Pos(),
				"%s.%s() in library code terminates the program; return an error instead",
				callee.Pkg().Name(), callee.Name())
			return
		}

		// Logger methods, whatever expression the logger comes from
		kind := "Fatal"
		if strings.HasPrefix(callee.Name(), "Panic") {
			kind = "Panic"
		}
		reporter.Reportf(call.Pos(),
			"%s log in library code terminates the program; return an error instead", kind)
	}
}
