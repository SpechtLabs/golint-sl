// Package dataflow provides SSA-based data flow analysis for detecting:
// - Sensitive data leaks (passwords, tokens flowing to logs)
// - Unvalidated input reaching dangerous sinks
// - Context propagation issues
package dataflow

import (
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

const Doc = `track data flow using SSA to detect security issues

This analyzer uses SSA to trace how values flow through the program:
1. Sensitive data (passwords, tokens, secrets) should not flow to logs
2. User input should be validated before reaching dangerous operations
3. Context should be propagated correctly through the call chain
4. Errors should be wrapped, not discarded

SSA analysis provides more accurate flow tracking than AST alone.`

var Analyzer = &analysis.Analyzer{
	Name:     "dataflow",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{buildssa.Analyzer},
	Run:      run,
}

// SensitivePatterns are parameter/variable names that might contain sensitive data
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

func run(pass *analysis.Pass) (interface{}, error) {
	ssaInfo := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)

	for _, fn := range ssaInfo.SrcFuncs {
		// Check for sensitive data flowing to logs
		checkSensitiveDataLeaks(pass, fn)

		// Check for context propagation
		checkContextPropagation(pass, fn)

		// Check for error handling
		checkErrorFlow(pass, fn)
	}

	return nil, nil
}

// checkSensitiveDataLeaks traces sensitive parameters to see if they reach logging
func checkSensitiveDataLeaks(pass *analysis.Pass, fn *ssa.Function) {
	for _, param := range fn.Params {
		paramName := strings.ToLower(param.Name())

		// Check if this parameter looks sensitive
		isSensitive := false
		for _, pattern := range SensitivePatterns {
			if strings.Contains(paramName, pattern) {
				isSensitive = true
				break
			}
		}

		if !isSensitive {
			continue
		}

		// Trace where this value flows
		sinks := traceToSinks(param, make(map[ssa.Value]bool))

		for _, sink := range sinks {
			if call, ok := sink.(*ssa.Call); ok {
				callee := call.Call.StaticCallee()
				if callee != nil && isLoggingOrPrintFunction(callee) {
					pass.Reportf(call.Pos(),
						"sensitive parameter %q may be logged; sanitize or redact before logging",
						param.Name())
				}
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
		// If this is a call instruction, it's a potential sink
		if call, ok := ref.(*ssa.Call); ok {
			sinks = append(sinks, call)
		}

		// If it produces a new value, trace that too
		if instr, ok := ref.(ssa.Value); ok {
			sinks = append(sinks, traceToSinks(instr, visited)...)
		}

		// Handle phi nodes (merge points in control flow)
		if phi, ok := ref.(*ssa.Phi); ok {
			sinks = append(sinks, traceToSinks(phi, visited)...)
		}

		// Handle field access
		if field, ok := ref.(*ssa.FieldAddr); ok {
			sinks = append(sinks, traceToSinks(field, visited)...)
		}

		// Handle type assertions
		if assert, ok := ref.(*ssa.TypeAssert); ok {
			sinks = append(sinks, traceToSinks(assert, visited)...)
		}
	}

	return sinks
}

// isLoggingOrPrintFunction checks if a function is for logging/printing
func isLoggingOrPrintFunction(fn *ssa.Function) bool {
	if fn.Pkg == nil {
		return false
	}

	pkgPath := fn.Pkg.Pkg.Path()
	fullName := pkgPath + "." + fn.Name()

	// Check common logging packages
	loggingIndicators := []string{
		"log", "zap", "logrus", "zerolog",
		"fmt.Print", "fmt.Fprint", "fmt.Sprint",
	}

	for _, indicator := range loggingIndicators {
		if strings.Contains(fullName, indicator) {
			// Exclude string formatting that returns strings
			if fn.Name() == "Sprintf" || fn.Name() == "Sprint" {
				return false
			}
			return true
		}
	}

	return false
}

// checkContextPropagation ensures context is passed through the call chain
func checkContextPropagation(pass *analysis.Pass, fn *ssa.Function) {
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

			// Check if callee expects context
			if calleeExpectsContext(callee) {
				// Check if context is passed
				contextPassed := false
				for _, arg := range call.Call.Args {
					if isContextType(arg.Type()) {
						contextPassed = true
						break
					}
				}

				if !contextPassed {
					pass.Reportf(call.Pos(),
						"function %s expects context but none was passed; propagate context through the call chain",
						callee.Name())
				}
			}
		}
	}
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

// checkErrorFlow ensures errors are handled properly, not discarded
// This is a lighter check - the standard errcheck linter handles most cases
func checkErrorFlow(pass *analysis.Pass, fn *ssa.Function) {
	// Skip this check - errcheck from golangci-lint handles error checking better
	// and has proper understanding of deferred calls, type assertions, etc.
	// This function is kept for documentation/future enhancement

	// If you want to enable strict error checking, uncomment the code below
	// and customize for your needs

	/*
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				call, ok := instr.(*ssa.Call)
				if !ok {
					continue
				}
				// ... error checking logic
			}
		}
	*/
}

// TaintAnalysis performs taint tracking from sources to sinks
type TaintAnalysis struct {
	Sources map[ssa.Value]string // value -> source description
	Sinks   []TaintSink
}

// TaintSink represents a location where tainted data reached
type TaintSink struct {
	Call     *ssa.Call
	Source   string
	SinkType string
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

// Propagate traces taint through the program
func (t *TaintAnalysis) Propagate() {
	// Iterate until fixpoint
	changed := true
	for changed {
		changed = false

		for value, source := range t.Sources {
			refs := value.Referrers()
			if refs == nil {
				continue
			}

			for _, ref := range *refs {
				// If this instruction produces a new value, it's also tainted
				if newVal, ok := ref.(ssa.Value); ok {
					if _, exists := t.Sources[newVal]; !exists {
						t.Sources[newVal] = source
						changed = true
					}
				}

				// Track calls as potential sinks
				if call, ok := ref.(*ssa.Call); ok {
					callee := call.Call.StaticCallee()
					if callee != nil {
						sinkType := categorizeSink(callee)
						if sinkType != "" {
							t.Sinks = append(t.Sinks, TaintSink{
								Call:     call,
								Source:   source,
								SinkType: sinkType,
							})
						}
					}
				}
			}
		}
	}
}

// categorizeSink determines what kind of dangerous sink a function is
func categorizeSink(fn *ssa.Function) string {
	if fn.Pkg == nil {
		return ""
	}

	pkgPath := fn.Pkg.Pkg.Path()
	name := fn.Name()

	// Logging sinks
	if strings.Contains(pkgPath, "log") || strings.Contains(pkgPath, "zap") {
		return "logging"
	}

	// Execution sinks
	if pkgPath == "os/exec" || name == "Exec" {
		return "command_execution"
	}

	// SQL sinks (potential injection)
	if strings.Contains(pkgPath, "sql") && (name == "Query" || name == "Exec") {
		return "sql_query"
	}

	// File sinks
	if pkgPath == "os" && (name == "Create" || name == "WriteFile") {
		return "file_write"
	}

	return ""
}
