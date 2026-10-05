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

// Doc is the exporteddoc analyzer's documentation.
const Doc = `ensure exported symbols have documentation comments

Exported functions, types, variables and constants should have
documentation. This enables godoc and IDE tooltips. The doc comment of a
function, a type, or a standalone var or const declaring one name should
start with the symbol name; in a parenthesized var or const group, a doc
comment on the group or on a spec is enough. Methods and Err-prefixed
sentinel errors are not checked.

Good:
    // Service handles business logic for user operations.
    type Service struct { ... }

    // ProcessRequest handles incoming API requests and returns results.
    func ProcessRequest(ctx context.Context, req *Request) (*Response, error)

Bad:
    type Service struct { ... }  // No documentation
    
    // handles requests  // Doesn't start with function name
    func ProcessRequest(...) ...`

// Analyzer reports exported symbols without documentation comments.
var Analyzer = &analysis.Analyzer{
	Name:     "exporteddoc",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Skip test files
	var inTestFile bool

	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.GenDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
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

	text := docText(fn.Doc)
	if text == "" {
		reporter.Reportf(fn.Pos(),
			"exported function %s should have a documentation comment",
			fn.Name.Name)
		return
	}

	// Check that doc starts with function name
	if !strings.HasPrefix(text, fn.Name.Name) {
		reporter.Reportf(fn.Doc.Pos(),
			"documentation for %s should start with %q",
			fn.Name.Name, fn.Name.Name)
	}
}

func checkGenDecl(reporter *nolint.Reporter, decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			checkTypeSpecDoc(reporter, decl, s)
		case *ast.ValueSpec:
			checkValueSpecDoc(reporter, decl, s)
		}
	}
}

// checkTypeSpecDoc checks the documentation of an exported type declared in decl.
func checkTypeSpecDoc(reporter *nolint.Reporter, decl *ast.GenDecl, s *ast.TypeSpec) {
	if !ast.IsExported(s.Name.Name) {
		return
	}

	// Check for documentation
	doc := s.Doc
	if doc == nil {
		doc = decl.Doc
	}

	text := docText(doc)
	if text == "" {
		reporter.Reportf(s.Pos(),
			"exported type %s should have a documentation comment",
			s.Name.Name)
		return
	}

	// Check that doc starts with type name
	if !strings.HasPrefix(text, s.Name.Name) {
		reporter.Reportf(doc.Pos(),
			"documentation for %s should start with %q",
			s.Name.Name, s.Name.Name)
	}
}

// checkValueSpecDoc checks that the exported variables and constants declared
// in decl are documented, and that the doc comment of a standalone
// declaration of one name starts with that name.
func checkValueSpecDoc(reporter *nolint.Reporter, decl *ast.GenDecl, s *ast.ValueSpec) {
	doc := s.Doc
	if doc == nil {
		doc = decl.Doc
	}
	text := docText(doc)

	for _, name := range s.Names {
		if !ast.IsExported(name.Name) {
			continue
		}

		// Skip error variables (Err*)
		if strings.HasPrefix(name.Name, "Err") {
			continue
		}

		if text == "" {
			reporter.Reportf(name.Pos(),
				"exported variable %s should have a documentation comment",
				name.Name)
			continue
		}

		// Only a standalone declaration of one name has a doc written for
		// that name. Inside parentheses, a comment often heads a section of
		// the group (as in an iota block), and a spec declaring several names
		// can't start with each of them.
		if !documentsOneName(decl, s) || strings.HasPrefix(text, name.Name) {
			continue
		}
		reporter.Reportf(doc.Pos(),
			"documentation for %s should start with %q",
			name.Name, name.Name)
	}
}

// documentsOneName reports whether s declares a single name outside
// parentheses, so that its doc comment is written for that name.
func documentsOneName(decl *ast.GenDecl, s *ast.ValueSpec) bool {
	return len(s.Names) == 1 && !decl.Lparen.IsValid()
}

// docText returns the text of a doc comment without its comment markers and
// directives such as //go:generate or //nolint:..., or "" if there is no doc
// comment or it holds only directives.
func docText(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}
	return strings.TrimSpace(doc.Text())
}
