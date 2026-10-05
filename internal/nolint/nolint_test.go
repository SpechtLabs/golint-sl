package nolint_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"

	"golang.org/x/tools/go/analysis"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// src has a directive or a plain comment on the line noted in each comment.
const src = `package p

//nolint:golint-sl
var a = 1 // line 4: suppressed for every analyzer by line 3

var b = 2 //nolint:contextfirst

var c = 3 // nolint:nilcheck,contextfirst

var d = 4 //nolint:nilcheck, contextfirst trailing words are ignored

var e = 5 //nolint:,,
/* nolint:contextfirst */
var f = 6 // not a nolint directive: nolint:contextfirst

var g = 7 //nolintcontextfirst
`

func parse(t *testing.T) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return fset, file
}

func TestFileDirectivesIsSuppressed(t *testing.T) {
	fset, file := parse(t)
	fd := nolint.ParseFile(file, fset)

	tests := []struct {
		name     string
		line     int
		analyzer string
		want     bool
	}{
		{name: "golint-sl directive on its own line", line: 3, analyzer: "contextfirst", want: true},
		{name: "golint-sl directive on the preceding line", line: 4, analyzer: "nilcheck", want: true},
		{name: "two lines below a directive", line: 5, analyzer: "nilcheck", want: false},
		{name: "inline directive for the analyzer", line: 6, analyzer: "contextfirst", want: true},
		{name: "inline directive for another analyzer", line: 6, analyzer: "nilcheck", want: false},
		{name: "directive also covers the next line", line: 7, analyzer: "contextfirst", want: true},
		{name: "space after slashes, first of a list", line: 8, analyzer: "nilcheck", want: true},
		{name: "space after slashes, second of a list", line: 8, analyzer: "contextfirst", want: true},
		{name: "list stops at the first space", line: 10, analyzer: "nilcheck", want: true},
		{name: "names after a space are not parsed", line: 10, analyzer: "contextfirst", want: false},
		{name: "only empty names suppress nothing", line: 12, analyzer: "contextfirst", want: false},
		{name: "block comment is not a directive", line: 13, analyzer: "contextfirst", want: false},
		{name: "directive must start the comment", line: 14, analyzer: "contextfirst", want: false},
		{name: "colon is required", line: 16, analyzer: "contextfirst", want: false},
		{name: "line without any comment", line: 1, analyzer: "contextfirst", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fd.IsSuppressed(tt.line, tt.analyzer); got != tt.want {
				t.Errorf("IsSuppressed(%d, %q) = %v, want %v", tt.line, tt.analyzer, got, tt.want)
			}
		})
	}
}

func TestFileDirectivesIsSuppressedNil(t *testing.T) {
	var fd *nolint.FileDirectives
	if fd.IsSuppressed(1, "contextfirst") {
		t.Error("nil FileDirectives must not suppress anything")
	}
}

func TestReporter(t *testing.T) {
	fset, file := parse(t)

	// lineStart returns the position of the first byte of the given line.
	lineStart := func(line int) token.Pos {
		return fset.File(file.Pos()).LineStart(line)
	}

	tests := []struct {
		name       string
		analyzer   string
		pos        func() token.Pos
		suppressed bool
	}{
		{name: "suppressed by golint-sl", analyzer: "contextfirst", pos: func() token.Pos { return lineStart(4) }, suppressed: true},
		{name: "suppressed by own name", analyzer: "contextfirst", pos: func() token.Pos { return lineStart(6) }, suppressed: true},
		{name: "not suppressed for another analyzer", analyzer: "nilcheck", pos: func() token.Pos { return lineStart(6) }, suppressed: false},
		{name: "line without directive", analyzer: "contextfirst", pos: func() token.Pos { return lineStart(1) }, suppressed: false},
		{name: "position outside the pass files", analyzer: "contextfirst", pos: func() token.Pos { return token.NoPos }, suppressed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, method := range []string{"Reportf", "Report"} {
				var got []analysis.Diagnostic
				pass := &analysis.Pass{
					Analyzer: &analysis.Analyzer{Name: tt.analyzer},
					Fset:     fset,
					Files:    []*ast.File{file},
					Report:   func(d analysis.Diagnostic) { got = append(got, d) },
				}
				r := nolint.NewReporter(pass)
				if r.AnalyzerName != tt.analyzer {
					t.Fatalf("AnalyzerName = %q, want %q", r.AnalyzerName, tt.analyzer)
				}
				if _, ok := r.Directives["p.go"]; !ok {
					t.Fatalf("Directives has no entry for p.go: %v", r.Directives)
				}

				pos := tt.pos()
				if method == "Reportf" {
					r.Reportf(pos, "found %d", 1)
				} else {
					r.Report(&analysis.Diagnostic{Pos: pos, Message: "found 1"})
				}

				var want []analysis.Diagnostic
				if !tt.suppressed {
					want = []analysis.Diagnostic{{Pos: pos, Message: "found 1"}}
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("%s: reported %+v, want %+v", method, got, want)
				}
			}
		})
	}
}
