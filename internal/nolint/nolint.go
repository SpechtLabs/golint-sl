// Package nolint provides support for suppressing linter diagnostics using
// special comments in the source code.
//
// Supported comment formats:
//
//	//nolint:golint-sl           - suppress all golint-sl analyzers on this line
//	//nolint:contextfirst        - suppress specific analyzer
//	//nolint:contextfirst,nilcheck - suppress multiple analyzers
//	// nolint:golint-sl          - space after // is allowed
//
// Comments can appear:
//   - On the same line as the code (inline)
//   - On the line immediately before the code
package nolint

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// nolintRegex matches nolint directives in comments.
// Matches: //nolint:name or // nolint:name or //nolint:name1,name2
var nolintRegex = regexp.MustCompile(`^//\s*nolint:([a-zA-Z0-9_,-]+)`)

// Directive represents a parsed nolint directive.
type Directive struct {
	Line      int      // Line number where the directive appears
	Analyzers []string // List of analyzer names to suppress (empty means all)
}

// FileDirectives holds all nolint directives for a file, indexed by line number.
type FileDirectives struct {
	// byLine maps line numbers to their directives.
	// A directive on line N applies to lines N and N+1 (for preceding comments).
	byLine map[int]*Directive
}

// ParseFile extracts all nolint directives from a file's comments.
func ParseFile(file *ast.File, fset *token.FileSet) *FileDirectives {
	fd := &FileDirectives{
		byLine: make(map[int]*Directive),
	}

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if d := parseComment(c.Text); d != nil {
				line := fset.Position(c.Pos()).Line
				d.Line = line
				fd.byLine[line] = d
			}
		}
	}

	return fd
}

// parseComment parses a single comment for nolint directive.
func parseComment(text string) *Directive {
	matches := nolintRegex.FindStringSubmatch(text)
	if matches == nil {
		return nil
	}

	// matches[1] contains the analyzer names
	names := strings.Split(matches[1], ",")

	// Clean up names
	var analyzers []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			analyzers = append(analyzers, name)
		}
	}

	return &Directive{
		Analyzers: analyzers,
	}
}

// IsSuppressed checks if a diagnostic at the given position should be suppressed
// for the specified analyzer.
func (fd *FileDirectives) IsSuppressed(line int, analyzerName string) bool {
	if fd == nil {
		return false
	}

	// Check the current line (inline comment)
	if d := fd.byLine[line]; d != nil {
		if d.matches(analyzerName) {
			return true
		}
	}

	// Check the previous line (preceding comment)
	if d := fd.byLine[line-1]; d != nil {
		if d.matches(analyzerName) {
			return true
		}
	}

	return false
}

// matches checks if the directive suppresses the given analyzer.
func (d *Directive) matches(analyzerName string) bool {
	for _, name := range d.Analyzers {
		// "golint-sl" suppresses all analyzers
		if name == "golint-sl" {
			return true
		}
		// Match specific analyzer name
		if name == analyzerName {
			return true
		}
	}
	return false
}

// Reporter wraps analysis.Pass to provide nolint-aware reporting.
type Reporter struct {
	Pass         *analysis.Pass
	Directives   map[string]*FileDirectives // filename -> directives
	AnalyzerName string
}

// NewReporter creates a new nolint-aware reporter for the given pass.
func NewReporter(pass *analysis.Pass) *Reporter {
	r := &Reporter{
		Pass:         pass,
		Directives:   make(map[string]*FileDirectives),
		AnalyzerName: pass.Analyzer.Name,
	}

	// Parse directives from all files in the package
	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		r.Directives[filename] = ParseFile(file, pass.Fset)
	}

	return r
}

// Reportf reports a diagnostic if it's not suppressed by a nolint directive.
func (r *Reporter) Reportf(pos token.Pos, format string, args ...interface{}) {
	position := r.Pass.Fset.Position(pos)

	// Check if this position is suppressed
	if fd := r.Directives[position.Filename]; fd != nil {
		if fd.IsSuppressed(position.Line, r.AnalyzerName) {
			return
		}
	}

	r.Pass.Reportf(pos, format, args...)
}

// Report reports a diagnostic if it's not suppressed by a nolint directive.
func (r *Reporter) Report(d *analysis.Diagnostic) {
	position := r.Pass.Fset.Position(d.Pos)

	// Check if this position is suppressed
	if fd := r.Directives[position.Filename]; fd != nil {
		if fd.IsSuppressed(position.Line, r.AnalyzerName) {
			return
		}
	}

	r.Pass.Report(*d)
}
