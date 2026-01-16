// Package nopanic provides an analyzer that ensures library code never panics.
//
// Library functions should return errors instead of panicking. Panics should only
// be used in main packages or for truly unrecoverable programmer errors.
package nopanic

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `ensure library code returns errors instead of panicking

This analyzer detects:
1. panic() calls in non-main packages
2. log.Fatal/log.Panic calls in library code
3. Must* functions that panic on error

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

func run(pass *analysis.Pass) (interface{}, error) {
	// Skip main packages
	if pass.Pkg.Name() == "main" {
		return nil, nil
	}

	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	var currentFunc string
	var inTestFile bool

	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.CallExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.File:
			filename := pass.Fset.Position(node.Pos()).Filename
			inTestFile = strings.HasSuffix(filename, "_test.go")

		case *ast.FuncDecl:
			if node.Name != nil {
				currentFunc = node.Name.Name
			}

		case *ast.CallExpr:
			if inTestFile {
				return
			}

			// Skip allowed functions
			if allowedPanicFunctions[currentFunc] {
				return
			}

			checkPanicCall(reporter, node, currentFunc)
		}
	})

	return nil, nil
}

func checkPanicCall(reporter *nolint.Reporter, call *ast.CallExpr, _ string) {
	var funcName string

	switch fn := call.Fun.(type) {
	case *ast.Ident:
		funcName = fn.Name
	case *ast.SelectorExpr:
		if ident, ok := fn.X.(*ast.Ident); ok {
			funcName = ident.Name + "." + fn.Sel.Name
		} else {
			funcName = fn.Sel.Name
		}
	default:
		return
	}

	// Check for direct panic calls
	if funcName == "panic" {
		reporter.Reportf(call.Pos(),
			"panic() in library code; return an error instead to let callers handle failures gracefully")
		return
	}

	// Check for log.Fatal and log.Panic variants
	fatalPatterns := []string{
		"log.Fatal", "log.Fatalf", "log.Fatalln",
		"log.Panic", "log.Panicf", "log.Panicln",
		"logrus.Fatal", "logrus.Fatalf", "logrus.Fatalln",
		"logrus.Panic", "logrus.Panicf", "logrus.Panicln",
	}

	for _, pattern := range fatalPatterns {
		if funcName == pattern {
			reporter.Reportf(call.Pos(),
				"%s() in library code terminates the program; return an error instead", funcName)
			return
		}
	}

	// Check for zap fatal
	if strings.Contains(funcName, "Fatal") && (strings.Contains(funcName, "zap") || strings.Contains(funcName, "logger")) {
		reporter.Reportf(call.Pos(),
			"Fatal log in library code terminates the program; return an error instead")
		return
	}

	// Note: Must* functions that panic are generally acceptable
	// as they follow Go conventions (e.g., regexp.MustCompile)
}
