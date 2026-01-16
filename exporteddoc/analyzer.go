// Package exporteddoc ensures exported symbols have documentation.
//
// Exported functions, types, and package-level variables should have
// documentation comments that explain their purpose.
package exporteddoc

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `ensure exported symbols have documentation comments

Exported functions, types, and variables should have documentation
that starts with the symbol name. This enables godoc and IDE tooltips.

Good:
    // Service handles business logic for user operations.
    type Service struct { ... }

    // ProcessRequest handles incoming API requests and returns results.
    func ProcessRequest(ctx context.Context, req *Request) (*Response, error)

Bad:
    type Service struct { ... }  // No documentation
    
    // handles requests  // Doesn't start with function name
    func ProcessRequest(...) ...`

var Analyzer = &analysis.Analyzer{
	Name:     "exporteddoc",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Skip test files
	var inTestFile bool

	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.GenDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		case *ast.File:
			filename := pass.Fset.Position(node.Pos()).Filename
			inTestFile = strings.HasSuffix(filename, "_test.go")

		case *ast.FuncDecl:
			if inTestFile {
				return
			}
			checkFuncDoc(reporter, node)

		case *ast.GenDecl:
			if inTestFile {
				return
			}
			checkGenDecl(reporter, node)
		}
	})

	return nil, nil
}

func checkFuncDoc(reporter *nolint.Reporter, fn *ast.FuncDecl) {
	// Only check exported functions
	if !ast.IsExported(fn.Name.Name) {
		return
	}

	// Skip methods - they're often self-explanatory
	if fn.Recv != nil {
		return
	}

	if fn.Doc == nil || len(fn.Doc.List) == 0 {
		reporter.Reportf(fn.Pos(),
			"exported function %s should have a documentation comment",
			fn.Name.Name)
		return
	}

	// Check that doc starts with function name
	firstLine := fn.Doc.List[0].Text
	if !strings.HasPrefix(firstLine, "// "+fn.Name.Name) {
		reporter.Reportf(fn.Doc.Pos(),
			"documentation for %s should start with %q",
			fn.Name.Name, fn.Name.Name)
	}
}

func checkGenDecl(reporter *nolint.Reporter, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			if !ast.IsExported(s.Name.Name) {
				continue
			}

			// Check for documentation
			doc := s.Doc
			if doc == nil {
				doc = decl.Doc
			}

			if doc == nil || len(doc.List) == 0 {
				reporter.Reportf(s.Pos(),
					"exported type %s should have a documentation comment",
					s.Name.Name)
				continue
			}

			// Check that doc starts with type name
			firstLine := doc.List[0].Text
			if !strings.HasPrefix(firstLine, "// "+s.Name.Name) {
				reporter.Reportf(doc.Pos(),
					"documentation for %s should start with %q",
					s.Name.Name, s.Name.Name)
			}

		case *ast.ValueSpec:
			// Check exported variables and constants
			for _, name := range s.Names {
				if !ast.IsExported(name.Name) {
					continue
				}

				// Skip error variables (Err*)
				if strings.HasPrefix(name.Name, "Err") {
					continue
				}

				doc := s.Doc
				if doc == nil {
					doc = decl.Doc
				}

				if doc == nil || len(doc.List) == 0 {
					reporter.Reportf(name.Pos(),
						"exported variable %s should have a documentation comment",
						name.Name)
				}
			}
		}
	}
}
