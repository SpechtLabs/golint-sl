// Package contextfirst ensures context.Context is always the first parameter.
//
// This is a Go convention that makes code consistent and easier to read.
// Context should flow through the entire call chain as the first argument.
package contextfirst

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the contextfirst analyzer's documentation.
const Doc = `ensure context.Context is always the first parameter

Go convention dictates that context.Context should be the first parameter
when a function accepts one. This makes the context flow obvious and consistent.
Only the standard library's context.Context counts; framework types named
Context, such as *gin.Context, are left alone.

Good:
    func ProcessRequest(ctx context.Context, req *Request) error
    func (s *Service) Handle(ctx context.Context, id string) (*Result, error)

Bad:
    func ProcessRequest(req *Request, ctx context.Context) error
    func (s *Service) Handle(id string, ctx context.Context) (*Result, error)

Reference: https://go.dev/blog/context#package-context`

// Analyzer reports functions whose context.Context parameter is not first.
var Analyzer = &analysis.Analyzer{
	Name:     "contextfirst",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		var params *ast.FieldList
		var name string
		var pos ast.Node

		switch node := n.(type) {
		case *ast.FuncDecl:
			params = node.Type.Params
			name = node.Name.Name
			pos = node
		case *ast.FuncLit:
			params = node.Type.Params
			name = "anonymous function"
			pos = node
		}

		if params == nil || params.NumFields() < 2 {
			return
		}

		// If a context exists but isn't first, report its position
		if ctxPos := contextParamPosition(pass, params); ctxPos > 0 {
			reporter.Reportf(pos.Pos(),
				"context.Context should be the first parameter in %s, not parameter %d",
				name, ctxPos+1)
		}
	})

	return nil, nil
}

// contextParamPosition returns the zero-based position of the first
// context.Context parameter in params, counting every name of a grouped field
// (a, b int counts as two parameters), or -1 when there is none.
func contextParamPosition(pass *analysis.Pass, params *ast.FieldList) int {
	pos := 0
	for _, field := range params.List {
		if isContextType(pass.TypesInfo.TypeOf(field.Type)) {
			return pos
		}
		pos += max(len(field.Names), 1)
	}
	return -1
}

// isContextType reports whether t is the standard library's context.Context.
// Framework types that happen to be called Context, such as *gin.Context, are
// not.
func isContextType(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context"
}
