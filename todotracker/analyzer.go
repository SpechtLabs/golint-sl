// Package todotracker ensures TODO/FIXME comments have owners and context.
//
// Orphaned TODOs tend to stay forever. Requiring ownership and context
// helps ensure technical debt is tracked and eventually addressed.
package todotracker

import (
	"go/ast"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `ensure TODO/FIXME comments have owners and context

Orphaned TODOs without owners tend to never get done. This analyzer
enforces that TODO/FIXME comments include:
1. An owner (username, email, or team)
2. Context about what needs to be done

Good:
    // TODO(username): Implement retry logic for transient failures
    // FIXME(@team-platform): This breaks when input exceeds 1MB
    // TODO(jira:PROJ-123): Add caching layer

Bad:
    // TODO: fix this
    // FIXME
    // TODO - make this better`

var Analyzer = &analysis.Analyzer{
	Name:     "todotracker",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Pattern to match well-formed TODOs: TODO(owner): description
var wellFormedTODO = regexp.MustCompile(`(?i)(TODO|FIXME)\s*\([^)]+\)\s*:\s*\S+`)

// Pattern to match any TODO/FIXME
var anyTODO = regexp.MustCompile(`(?i)(TODO|FIXME)`)

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.File)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		file := n.(*ast.File)

		for _, cg := range file.Comments {
			for _, comment := range cg.List {
				checkComment(reporter, comment)
			}
		}
	})

	return nil, nil
}

func checkComment(reporter *nolint.Reporter, comment *ast.Comment) {
	text := comment.Text

	// Check if this comment contains TODO or FIXME
	if !anyTODO.MatchString(text) {
		return
	}

	// Check if it's well-formed
	if wellFormedTODO.MatchString(text) {
		return // Good TODO
	}

	// Extract the TODO/FIXME part for the message
	todoType := "TODO"
	if strings.Contains(strings.ToUpper(text), "FIXME") {
		todoType = "FIXME"
	}

	// Determine what's wrong
	if !strings.Contains(text, "(") {
		reporter.Reportf(comment.Pos(),
			"%s without owner; use %s(username): description",
			todoType, todoType)
	} else if !strings.Contains(text, ":") {
		reporter.Reportf(comment.Pos(),
			"%s without description; use %s(owner): what needs to be done",
			todoType, todoType)
	} else {
		// Has parens and colon but doesn't match pattern - likely malformed
		reporter.Reportf(comment.Pos(),
			"%s appears malformed; use format: %s(owner): description",
			todoType, todoType)
	}
}
