// Package contextfirst ensures context.Context is always the first parameter.
//
// This is a Go convention that makes code consistent and easier to read.
// Context should flow through the entire call chain as the first argument.
package contextfirst

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `ensure context.Context is always the first parameter

Go convention dictates that context.Context should be the first parameter
when a function accepts one. This makes the context flow obvious and consistent.

Good:
    func ProcessRequest(ctx context.Context, req *Request) error
    func (s *Service) Handle(ctx context.Context, id string) (*Result, error)

Bad:
    func ProcessRequest(req *Request, ctx context.Context) error
    func (s *Service) Handle(id string, ctx context.Context) (*Result, error)

Reference: https://go.dev/blog/context#package-context`

var Analyzer = &analysis.Analyzer{
	Name:     "contextfirst",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
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

		if params == nil || len(params.List) < 2 {
			return
		}

		// Find context parameter position
		ctxPos := -1
		for i, field := range params.List {
			if isContextType(field.Type) {
				ctxPos = i
				break
			}
		}

		// If context exists but isn't first, report
		if ctxPos > 0 {
			pass.Reportf(pos.Pos(),
				"context.Context should be the first parameter in %s, not parameter %d",
				name, ctxPos+1)
		}
	})

	return nil, nil
}

func isContextType(expr ast.Expr) bool {
	typeStr := types.ExprString(expr)
	return typeStr == "context.Context" || strings.HasSuffix(typeStr, ".Context")
}
