// Package todotracker ensures TODO/FIXME comments have owners and context.
//
// Orphaned TODOs tend to stay forever. Requiring ownership and context
// helps ensure technical debt is tracked and eventually addressed.
package todotracker

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the todotracker analyzer's documentation.
const Doc = `ensure TODO/FIXME comments have owners and context

Orphaned TODOs without owners tend to never get done. This analyzer
enforces that TODO/FIXME comments include:
1. An owner (username, email, or team)
2. Context about what needs to be done

A marker is the upper-case word TODO or FIXME at the start of a comment
line. Words that merely contain the letters (Mastodon), lower-case prose
and identifiers such as context.TODO are not markers.

Good:
    // TODO(username): Implement retry logic for transient failures
    // FIXME(@team-platform): This breaks when input exceeds 1MB
    // TODO(jira:PROJ-123): Add caching layer

Bad:
    // TODO: fix this
    // FIXME
    // TODO - make this better`

// Analyzer reports TODO and FIXME comments without an owner or context.
var Analyzer = &analysis.Analyzer{
	Name:     "todotracker",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// markerLine matches a comment line that starts with a marker: the
// upper-case word TODO or FIXME, not followed by another word character.
var markerLine = regexp.MustCompile(`^(TODO|FIXME)\b`)

// wellFormed matches a marker line of the form TODO(owner): description.
var wellFormed = regexp.MustCompile(`^(TODO|FIXME)\s*\([^)]+\)\s*:\s*\S`)

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.File)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		file := n.(*ast.File)

		for _, cg := range file.Comments {
			for _, comment := range cg.List {
				checkComment(reporter, comment)
			}
		}
	})

	return nil, nil
}

// checkComment reports every marker line of comment that lacks an owner or
// a description.
func checkComment(reporter *nolint.Reporter, comment *ast.Comment) {
	// The body starts after "//" or "/*", both two bytes long.
	body, offset := commentBody(comment.Text), 2

	for line := range strings.SplitSeq(body, "\n") {
		trimmed, indent := trimLinePrefix(line)
		if marker := markerLine.FindString(trimmed); marker != "" {
			checkMarker(reporter, comment.Pos()+token.Pos(offset+indent), marker, trimmed)
		}
		offset += len(line) + 1
	}
}

// commentBody strips the comment markers from text, so that the closing */
// of a block comment never reads as a description.
func commentBody(text string) string {
	if body, ok := strings.CutPrefix(text, "/*"); ok {
		return strings.TrimSuffix(body, "*/")
	}

	return strings.TrimPrefix(text, "//")
}

// trimLinePrefix removes the leading white space of a comment line and, for
// the continuation lines of a block comment, a leading "*". It returns the
// trimmed line and the number of bytes removed.
func trimLinePrefix(line string) (string, int) {
	trimmed := strings.TrimLeft(line, " \t")
	if rest, ok := strings.CutPrefix(trimmed, "*"); ok {
		trimmed = strings.TrimLeft(rest, " \t")
	}

	return trimmed, len(line) - len(trimmed)
}

// checkMarker reports the marker line text at pos unless it has the form
// marker(owner): description.
func checkMarker(reporter *nolint.Reporter, pos token.Pos, marker, text string) {
	if wellFormed.MatchString(text) {
		return
	}

	// Determine what's wrong
	switch {
	case !strings.Contains(text, "("):
		reporter.Reportf(pos,
			"%s without owner; use %s(username): description",
			marker, marker)
	case !strings.Contains(text, ":"):
		reporter.Reportf(pos,
			"%s without description; use %s(owner): what needs to be done",
			marker, marker)
	default:
		// Has parens and colon but doesn't match pattern - likely malformed
		reporter.Reportf(pos,
			"%s appears malformed; use format: %s(owner): description",
			marker, marker)
	}
}
