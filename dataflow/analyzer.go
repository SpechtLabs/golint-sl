// Package dataflow provides SSA-based data flow analysis for detecting:
// - Sensitive data leaks (passwords, tokens flowing to logs)
// - Context propagation issues
//
// It also exports a small taint tracker (TaintAnalysis) for tracing values
// to dangerous sinks.
package dataflow

import (
	"cmp"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"

	"github.com/spechtlabs/golint-sl/internal/credname"
	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the dataflow analyzer's documentation.
const Doc = `track data flow using SSA to detect security issues

This analyzer uses SSA to trace how values flow through the program:
1. Sensitive parameters, those whose name names a credential (dbPassword,
   authToken, apiKey; not key, author or secretName), should not flow to
   logging or printing functions, directly or as variadic arguments
2. A function that has a context should pass one to callees whose first
   parameter is a context

SSA analysis provides more accurate flow tracking than AST alone.`

// Analyzer tracks data flow through SSA to report security issues.
var Analyzer = &analysis.Analyzer{
	Name:     "dataflow",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{buildssa.Analyzer},
	Run:      run,
}

// SensitivePatterns are the credential words parameter names used to be
// matched against as substrings, which made key, auth and cert match monkey,
// author and certainty.
//
// Deprecated: the analyzer now decides by the words of a parameter's name
// (see internal/credname): dbPassword, authToken and apiKey are sensitive,
// key, author and tokenizer are not. The list is kept for compatibility and
// is not used.
var SensitivePatterns = []string{
	"password", "passwd", "pwd",
	"secret", "token", "key",
	"credential", "cred",
	"auth", "apikey", "api_key",
	"private", "cert", "certificate",
}

// DangerousSinks are functions that should not receive unvalidated/sensitive data
var DangerousSinks = []string{
	"log.Print", "log.Printf", "log.Println",
	"fmt.Print", "fmt.Printf", "fmt.Println",
	"zap.String", "zap.Any", // Unless properly sanitized
	"os.Exec", "exec.Command",
	"sql.Query", "sql.Exec", // SQL injection risk
}

// loggingPathSegments are import path elements that mark a logging package:
// log, log/slog, go.uber.org/zap, github.com/sirupsen/logrus,
// github.com/rs/zerolog and the like, or a project's own logging package.
var loggingPathSegments = map[string]bool{
	"log":     true,
	"slog":    true,
	"logging": true,
	"logger":  true,
	"zap":     true,
	"logrus":  true,
	"zerolog": true,
	"logr":    true,
	"klog":    true,
	"otelzap": true,
}

// fmtPrintFunctions are the fmt functions that write their arguments out.
// Sprint, Sprintf, Sprintln and Errorf only build values.
var fmtPrintFunctions = map[string]bool{
	"Print": true, "Printf": true, "Println": true,
	"Fprint": true, "Fprintf": true, "Fprintln": true,
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	ssaInfo := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)

	for _, fn := range ssaInfo.SrcFuncs {
		// Check for sensitive data flowing to logs
		checkSensitiveDataLeaks(reporter, fn)

		// Check for context propagation
		checkContextPropagation(reporter, fn)
	}

	return nil, nil
}

// checkSensitiveDataLeaks traces sensitive parameters to see if they reach logging
func checkSensitiveDataLeaks(reporter *nolint.Reporter, fn *ssa.Function) {
	for _, param := range fn.Params {
		// Check if this parameter names a credential
		if !credname.IsCredential(param.Name()) {
			continue
		}

		// Trace where this value flows
		sinks := traceToSinks(param, make(map[ssa.Value]bool))

		for _, sink := range sinks {
			call, ok := sink.(*ssa.Call)
			if !ok {
				continue
			}
			if callee := call.Call.StaticCallee(); callee != nil && isLoggingOrPrintFunction(callee) {
				reporter.Reportf(call.Pos(),
					"sensitive parameter %q may be logged; sanitize or redact before logging",
					param.Name())
			}
		}
	}
}

// traceToSinks follows a value through the SSA graph to find where it's used
func traceToSinks(value ssa.Value, visited map[ssa.Value]bool) []ssa.Instruction {
	if visited[value] {
		return nil
	}
	visited[value] = true

	var sinks []ssa.Instruction

	refs := value.Referrers()
	if refs == nil {
		return sinks
	}

	for _, ref := range *refs {
		switch instr := ref.(type) {
		case *ssa.Call:
			// A call is a potential sink, and its result is traced further
			sinks = append(sinks, instr)
			sinks = append(sinks, traceToSinks(instr, visited)...)
		case ssa.Value:
			// Any other instruction that produces a value (conversions, phi
			// nodes, field access, type assertions) carries the value on
			sinks = append(sinks, traceToSinks(instr, visited)...)
		case *ssa.Store:
			// Storing the value into an element of a local array is how
			// variadic arguments (fmt.Println(password)) and slice literals
			// are built; the array then flows on as a slice
			if array := storedArray(instr, value); array != nil {
				sinks = append(sinks, traceToSinks(array, visited)...)
			}
		}
	}

	return sinks
}

// storedArray returns the local array that store writes value into an element
// of, or nil when store does something else.
func storedArray(store *ssa.Store, value ssa.Value) *ssa.Alloc {
	if store.Val != value {
		return nil
	}
	elem, ok := store.Addr.(*ssa.IndexAddr)
	if !ok {
		return nil
	}
	alloc, ok := elem.X.(*ssa.Alloc)
	if !ok {
		return nil
	}
	return alloc
}

// isLoggingOrPrintFunction checks if a function is for logging/printing
func isLoggingOrPrintFunction(fn *ssa.Function) bool {
	if fn.Pkg == nil {
		return false
	}

	pkgPath := fn.Pkg.Pkg.Path()
	if pkgPath == "fmt" {
		return fmtPrintFunctions[fn.Name()]
	}
	return isLoggingPackage(pkgPath)
}

// isLoggingPackage reports whether one of pkgPath's elements names a logging
// package. Whole elements only: catalog and dialog are not logging packages.
func isLoggingPackage(pkgPath string) bool {
	for segment := range strings.SplitSeq(pkgPath, "/") {
		if loggingPathSegments[segment] {
			return true
		}
	}
	return false
}

// checkContextPropagation ensures context is passed through the call chain
func checkContextPropagation(reporter *nolint.Reporter, fn *ssa.Function) {
	// Check if function accepts context
	hasContextParam := false
	for _, param := range fn.Params {
		if isContextType(param.Type()) {
			hasContextParam = true
			break
		}
	}

	if !hasContextParam {
		return
	}

	// Check all calls within the function
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(*ssa.Call)
			if !ok {
				continue
			}

			callee := call.Call.StaticCallee()
			if callee == nil {
				continue
			}

			// Flag callees that expect a context but are not passed one
			if calleeExpectsContext(callee) && !passesContext(call) {
				reporter.Reportf(call.Pos(),
					"function %s expects context but none was passed; propagate context through the call chain",
					callee.Name())
			}
		}
	}
}

// passesContext checks if any argument of call is a context
func passesContext(call *ssa.Call) bool {
	for _, arg := range call.Call.Args {
		if isContextType(arg.Type()) {
			return true
		}
	}
	return false
}

// isContextType checks if a type is context.Context
func isContextType(t types.Type) bool {
	return strings.Contains(t.String(), "context.Context")
}

// calleeExpectsContext checks if a function's first parameter is context
func calleeExpectsContext(fn *ssa.Function) bool {
	if fn.Signature == nil {
		return false
	}

	params := fn.Signature.Params()
	if params.Len() == 0 {
		return false
	}

	firstParam := params.At(0)
	return isContextType(firstParam.Type())
}

// TaintAnalysis performs taint tracking from sources to sinks
type TaintAnalysis struct {
	Sources map[ssa.Value]string // value -> source description

	// recorded holds the sinks already in Sinks, so a call reached from the
	// same source along several paths is recorded once
	recorded map[sinkKey]bool

	Sinks []TaintSink
}

// TaintSink represents a location where tainted data reached
type TaintSink struct {
	Call     *ssa.Call
	Source   string
	SinkType string
}

// sinkKey identifies a recorded sink.
type sinkKey struct {
	call   *ssa.Call
	source string
}

// NewTaintAnalysis creates a new taint analysis tracker
func NewTaintAnalysis() *TaintAnalysis {
	return &TaintAnalysis{
		Sources: make(map[ssa.Value]string),
	}
}

// MarkSource marks a value as tainted from a particular source
func (t *TaintAnalysis) MarkSource(value ssa.Value, source string) {
	t.Sources[value] = source
}

// Propagate traces taint through the program. Every tainted value is visited
// once, and each call is recorded as a sink at most once per source. A value
// reachable from several sources is attributed to the first one in order of
// source description and position, so the result does not depend on map
// iteration order.
func (t *TaintAnalysis) Propagate() {
	worklist := make([]ssa.Value, 0, len(t.Sources))
	for value := range t.Sources {
		worklist = append(worklist, value)
	}
	slices.SortFunc(worklist, func(a, b ssa.Value) int {
		return cmp.Or(
			strings.Compare(t.Sources[a], t.Sources[b]),
			cmp.Compare(a.Pos(), b.Pos()),
			strings.Compare(a.Name(), b.Name()),
		)
	})

	for len(worklist) > 0 {
		value := worklist[0]
		worklist = worklist[1:]
		worklist = append(worklist, t.propagateFrom(value, t.Sources[value])...)
	}
}

// propagateFrom taints the values that value's referrers produce and records
// the sinks it reaches; it returns the values it newly tainted
func (t *TaintAnalysis) propagateFrom(value ssa.Value, source string) []ssa.Value {
	refs := value.Referrers()
	if refs == nil {
		return nil
	}

	var tainted []ssa.Value
	for _, ref := range *refs {
		// If this instruction produces a new value, it's also tainted
		if newVal, ok := ref.(ssa.Value); ok {
			if _, exists := t.Sources[newVal]; !exists {
				t.Sources[newVal] = source
				tainted = append(tainted, newVal)
			}
		}

		// Track calls as potential sinks
		if call, ok := ref.(*ssa.Call); ok {
			t.recordSink(call, source)
		}
	}

	return tainted
}

// recordSink records call as a sink of source if its callee is a dangerous
// sink and it isn't recorded yet
func (t *TaintAnalysis) recordSink(call *ssa.Call, source string) {
	callee := call.Call.StaticCallee()
	if callee == nil {
		return
	}

	sinkType := categorizeSink(callee)
	if sinkType == "" {
		return
	}

	key := sinkKey{call: call, source: source}
	if t.recorded[key] {
		return
	}
	if t.recorded == nil {
		t.recorded = make(map[sinkKey]bool)
	}
	t.recorded[key] = true

	t.Sinks = append(t.Sinks, TaintSink{
		Call:     call,
		Source:   source,
		SinkType: sinkType,
	})
}

// categorizeSink determines what kind of dangerous sink a function is
func categorizeSink(fn *ssa.Function) string {
	if fn.Pkg == nil {
		return ""
	}

	pkgPath := fn.Pkg.Pkg.Path()
	name := fn.Name()

	// Logging sinks
	if isLoggingPackage(pkgPath) {
		return "logging"
	}

	// SQL sinks (potential injection), before the generic Exec check below
	// so (*sql.DB).Exec is a query, not a command
	if strings.Contains(pkgPath, "sql") && (name == "Query" || name == "Exec") {
		return "sql_query"
	}

	// Execution sinks
	if pkgPath == "os/exec" || name == "Exec" {
		return "command_execution"
	}

	// File sinks
	if pkgPath == "os" && (name == "Create" || name == "WriteFile") {
		return "file_write"
	}

	return ""
}
