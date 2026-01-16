// Package wideevents provides an analyzer that enforces wide event logging patterns
// instead of traditional scattered log statements.
//
// Based on the philosophy from https://loggingsucks.com/
//
// Wide events are single, context-rich log events emitted per request per service,
// containing all relevant information for debugging. Instead of 15 log lines for
// one request, emit 1 line with 50+ structured fields.
//
// This analyzer:
// - Bans traditional loggers (logrus, stdlib log, fmt.Print)
// - Standardizes on zap for structured logging
// - Detects scattered log statements (multiple logs per function)
// - Enforces structured fields over string messages
// - Integrates with OpenTelemetry/Datadog span attributes
package wideevents

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

const Doc = `enforce wide event logging patterns instead of traditional logging

This analyzer implements the "logging sucks" philosophy (https://loggingsucks.com/):

1. BANS traditional loggers:
   - logrus.* (use zap instead)
   - log.* from stdlib (use zap instead)
   - fmt.Print/Printf/Println (use zap.Debug for dev output)

2. ENFORCES structured logging with zap:
   - Require structured fields (zap.String, zap.Int, etc.)
   - Flag bare string messages without context
   - Suggest using span attributes for tracing

3. ENFORCES span attributes when context is available:
   - If a function has context.Context, use trace.SpanFromContext(ctx) to get the span
   - Add wide-event attributes to the span via span.SetAttributes()
   - Span attributes provide better observability than logs alone

4. DETECTS anti-patterns:
   - Multiple log statements in a single function (should be one wide event)
   - Info/Warn/Error logs without request context (trace_id, request_id, user_id)
   - Logging inside loops (creates log spam)
   - Functions with context that log but don't set span attributes

5. ALLOWS:
   - zap.Debug for development/troubleshooting
   - Single wide event emission at function end
   - Span attributes for OpenTelemetry integration

The goal: One log line per request per service with all necessary context,
not scattered log statements throughout your code. When you have a context,
add attributes to the span for better distributed tracing.`

var Analyzer = &analysis.Analyzer{
	Name:     "wideevents",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Banned logging patterns - these should not be used
var bannedLogPatterns = map[string]string{
	// logrus - banned entirely
	"logrus.Info":       "logrus is banned; use zap with structured fields for wide events",
	"logrus.Infof":      "logrus is banned; use zap with structured fields for wide events",
	"logrus.Warn":       "logrus is banned; use zap with structured fields for wide events",
	"logrus.Warnf":      "logrus is banned; use zap with structured fields for wide events",
	"logrus.Error":      "logrus is banned; use zap with structured fields for wide events",
	"logrus.Errorf":     "logrus is banned; use zap with structured fields for wide events",
	"logrus.Fatal":      "logrus is banned; use zap with structured fields for wide events",
	"logrus.Fatalf":     "logrus is banned; use zap with structured fields for wide events",
	"logrus.Debug":      "logrus is banned; use zap.Debug with structured fields instead",
	"logrus.Debugf":     "logrus is banned; use zap.Debug with structured fields instead",
	"logrus.WithField":  "logrus is banned; use zap with structured fields for wide events",
	"logrus.WithFields": "logrus is banned; use zap with structured fields for wide events",

	// stdlib log - banned entirely
	"log.Print":   "stdlib log is banned; use zap with structured fields for wide events",
	"log.Printf":  "stdlib log is banned; use zap with structured fields for wide events",
	"log.Println": "stdlib log is banned; use zap with structured fields for wide events",
	"log.Fatal":   "stdlib log is banned; use zap.Fatal with structured fields instead",
	"log.Fatalf":  "stdlib log is banned; use zap.Fatal with structured fields instead",
	"log.Fatalln": "stdlib log is banned; use zap.Fatal with structured fields instead",
	"log.Panic":   "stdlib log is banned; use zap.Panic with structured fields instead",
	"log.Panicf":  "stdlib log is banned; use zap.Panic with structured fields instead",
	"log.Panicln": "stdlib log is banned; use zap.Panic with structured fields instead",

	// fmt.Print - banned for logging (use for CLI output only)
	"fmt.Print":   "fmt.Print is not for logging; use zap.Debug for dev output or emit a wide event",
	"fmt.Printf":  "fmt.Printf is not for logging; use zap.Debug for dev output or emit a wide event",
	"fmt.Println": "fmt.Println is not for logging; use zap.Debug for dev output or emit a wide event",
}

// Traditional logging methods that should be replaced with wide events
var traditionalLogMethods = map[string]bool{
	"Info":   true,
	"Infof":  true,
	"Infow":  true,
	"Warn":   true,
	"Warnf":  true,
	"Warnw":  true,
	"Error":  true,
	"Errorf": true,
	"Errorw": true,
}

// Debug methods are allowed (for development)
var allowedDebugMethods = map[string]bool{
	"Debug":  true,
	"Debugf": true,
	"Debugw": true,
}

// Required context fields for wide events
var requiredContextFields = []string{
	"request_id",
	"trace_id",
	"span_id",
	"user_id",
	"service",
}

// Span-related function names for OpenTelemetry
var spanFromContextFuncs = map[string]bool{
	"SpanFromContext": true, // trace.SpanFromContext
	"Start":           true, // tracer.Start (creates new span)
	"StartSpan":       true, // common pattern
}

var spanSetAttributesMethods = map[string]bool{
	"SetAttributes": true,
	"SetAttribute":  true, // some APIs use singular
	"AddEvent":      true, // span events are also valid
	"SetStatus":     true, // setting status is also valid span usage
}

func run(pass *analysis.Pass) (interface{}, error) {
	reporter := nolint.NewReporter(pass)
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return
		}

		// Get file path for context-aware checks
		pos := pass.Fset.Position(fn.Pos())
		filePath := pos.Filename

		// Skip test files entirely
		if strings.HasSuffix(filePath, "_test.go") {
			return
		}

		// Skip test functions
		if fn.Name != nil && strings.HasPrefix(fn.Name.Name, "Test") {
			return
		}

		// Skip init and main
		if fn.Name != nil && (fn.Name.Name == "init" || fn.Name.Name == "main") {
			return
		}

		// Determine if this is a CLI package (cmd/ directory or cli/ directory)
		isCLIPackage := strings.Contains(filePath, "/cmd/") || strings.Contains(filePath, "/cli/")

		checkFunction(reporter, fn, isCLIPackage)
	})

	return nil, nil
}

func checkFunction(reporter *nolint.Reporter, fn *ast.FuncDecl, isCLIPackage bool) {
	var logCalls []*logCallInfo
	var logsInLoops []*ast.CallExpr

	// Check if function has a context parameter
	hasContext := functionHasContext(fn)
	hasSpanUsage := false
	hasSpanAttributes := false

	// Collect all log calls and span usage in the function
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		// Track if we're inside a loop
		switch node := n.(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			// Check for log calls inside this loop
			ast.Inspect(node, func(inner ast.Node) bool {
				if call, ok := inner.(*ast.CallExpr); ok {
					if info := analyzeLogCall(call); info != nil {
						logsInLoops = append(logsInLoops, call)
					}
				}
				return true
			})
			return false // Don't recurse again

		case *ast.CallExpr:
			// Check banned patterns first (but allow fmt.Print* in CLI packages)
			checkBannedLogPatterns(reporter, node, isCLIPackage)

			// Check for span usage
			if isSpanFromContextCall(node) {
				hasSpanUsage = true
			}
			if isSpanSetAttributesCall(node) {
				hasSpanAttributes = true
			}

			// Analyze the log call
			if info := analyzeLogCall(node); info != nil {
				logCalls = append(logCalls, info)
			}
		}
		return true
	})

	// Report logs inside loops
	for _, call := range logsInLoops {
		reporter.Reportf(call.Pos(),
			"logging inside loop creates log spam; accumulate data and emit one wide event after the loop")
	}

	// Check for scattered log statements (multiple non-debug logs)
	nonDebugLogs := 0
	for _, info := range logCalls {
		if !info.isDebug {
			nonDebugLogs++
		}
	}

	if nonDebugLogs > 1 {
		reporter.Reportf(fn.Pos(),
			"function has %d log statements; consider emitting a single wide event at the end instead of scattered logs",
			nonDebugLogs)
	}

	// Check each log call for required context
	for _, info := range logCalls {
		if !info.isDebug && !info.hasStructuredFields {
			reporter.Reportf(info.call.Pos(),
				"log call without structured fields; use zap.String(\"field\", value) to add context for wide events")
		}

		// Check for traditional log methods that should be wide events
		if info.isTraditionalLog && !info.isDebug {
			checkWideEventContext(reporter, info)
		}
	}

	// If function has context and logs but doesn't use span attributes, suggest it
	if hasContext && len(logCalls) > 0 && !hasSpanAttributes {
		// Only report if there are non-debug logs
		hasNonDebugLogs := false
		for _, info := range logCalls {
			if !info.isDebug {
				hasNonDebugLogs = true
				break
			}
		}

		if hasNonDebugLogs {
			if !hasSpanUsage {
				reporter.Reportf(fn.Pos(),
					"function has context.Context but doesn't use span attributes; "+
						"use span := trace.SpanFromContext(ctx) and span.SetAttributes() for better observability")
			} else if !hasSpanAttributes {
				reporter.Reportf(fn.Pos(),
					"function gets span from context but doesn't set attributes; "+
						"add span.SetAttributes(attribute.String(\"key\", value)) for wide event data")
			}
		}
	}
}

type logCallInfo struct {
	call                *ast.CallExpr
	method              string
	isDebug             bool
	isTraditionalLog    bool
	hasStructuredFields bool
	fieldNames          []string
}

// zapFieldMethods are methods on the zap package that return zap.Field, not log calls
var zapFieldMethods = map[string]bool{
	"String":     true,
	"Int":        true,
	"Int64":      true,
	"Int32":      true,
	"Float64":    true,
	"Float32":    true,
	"Bool":       true,
	"Duration":   true,
	"Time":       true,
	"Error":      true,
	"NamedError": true,
	"Any":        true,
	"Object":     true,
	"Array":      true,
	"Binary":     true,
	"ByteString": true,
	"Reflect":    true,
	"Stack":      true,
	"Stringer":   true,
	"Uint":       true,
	"Uint64":     true,
	"Uint32":     true,
}

func analyzeLogCall(call *ast.CallExpr) *logCallInfo {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	method := sel.Sel.Name

	// Skip fmt.Errorf/Sprintf - it's error/string construction, not logging
	if ident, ok := sel.X.(*ast.Ident); ok {
		if ident.Name == "fmt" && (method == "Errorf" || method == "Sprintf") {
			return nil
		}
		// Skip zap.String(), zap.Error(), etc. - these are field constructors, not log calls
		if ident.Name == "zap" && zapFieldMethods[method] {
			return nil
		}
	}

	// Check if this is a zap logger call
	isZapCall := false
	isLoggerMethod := false

	// Check for logger.Info(), logger.Error(), etc.
	if traditionalLogMethods[method] || allowedDebugMethods[method] {
		isLoggerMethod = true
		// Check if receiver looks like a logger
		switch x := sel.X.(type) {
		case *ast.Ident:
			name := strings.ToLower(x.Name)
			// Exclude fmt package - fmt.Errorf is not logging
			if name == "fmt" {
				return nil
			}
			// Exclude testing.T methods - t.Errorf is not logging
			if name == "t" || name == "b" {
				return nil
			}
			// Exclude error variables - err.Error(), e.Error(), herr.Error(), lastCause.Error() is not logging
			if name == "err" || name == "e" || strings.Contains(name, "err") || strings.Contains(name, "cause") {
				return nil
			}
			// Exclude common test assertion variables
			if name == "got" || name == "want" || name == "expected" || name == "actual" {
				return nil
			}
			if strings.Contains(name, "log") || strings.Contains(name, "logger") || name == "l" {
				isZapCall = true
			}
		case *ast.SelectorExpr:
			// Could be pkg.Logger or obj.logger, or struct.err.Error()
			if x.Sel != nil {
				fieldName := strings.ToLower(x.Sel.Name)
				// Exclude struct fields that are errors (e.g., event.err.Error())
				if fieldName == "err" || strings.Contains(fieldName, "error") {
					return nil
				}
				if strings.Contains(fieldName, "log") || strings.Contains(fieldName, "logger") {
					isZapCall = true
				}
			}
		case *ast.CallExpr:
			// Could be zap.L().Info() or similar
			isZapCall = true
		}
	}

	// Check for zap.L().Info() pattern
	if callExpr, ok := sel.X.(*ast.CallExpr); ok {
		if innerSel, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := innerSel.X.(*ast.Ident); ok {
				if ident.Name == "zap" && (innerSel.Sel.Name == "L" || innerSel.Sel.Name == "S") {
					isZapCall = true
				}
			}
		}
	}

	if !isLoggerMethod {
		return nil
	}

	// Logger calls must have at least one argument (the message)
	// err.Error() with no arguments is NOT a log call - it's the error interface method
	if len(call.Args) == 0 {
		return nil
	}

	info := &logCallInfo{
		call:             call,
		method:           method,
		isDebug:          allowedDebugMethods[method],
		isTraditionalLog: traditionalLogMethods[method],
	}

	// Check for structured fields in arguments
	info.hasStructuredFields, info.fieldNames = hasStructuredFields(call)

	// Only return if this looks like a logging call
	if isZapCall || isLoggerMethod {
		return info
	}

	return nil
}

func hasStructuredFields(call *ast.CallExpr) (bool, []string) {
	var fieldNames []string

	// Skip the first argument (message string)
	for i, arg := range call.Args {
		if i == 0 {
			continue // Skip message
		}

		// Check for zap.String(), zap.Int(), zap.Error(), etc.
		if argCall, ok := arg.(*ast.CallExpr); ok {
			if sel, ok := argCall.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					if ident.Name == "zap" {
						// zap.Error() is a special case - the field name is "error"
						if sel.Sel.Name == "Error" || sel.Sel.Name == "NamedError" {
							fieldNames = append(fieldNames, "error")
							continue
						}
						// Extract field name if possible
						if len(argCall.Args) > 0 {
							if lit, ok := argCall.Args[0].(*ast.BasicLit); ok {
								fieldNames = append(fieldNames, strings.Trim(lit.Value, "\""))
							}
						}
					}
				}
			}
		}
	}

	return len(fieldNames) > 0, fieldNames
}

func checkBannedLogPatterns(reporter *nolint.Reporter, call *ast.CallExpr, isCLIPackage bool) {
	callName := getCallName(call)
	if callName == "" {
		return
	}

	// Allow fmt.Print* in CLI packages - these are user output, not logging
	if isCLIPackage && (callName == "fmt.Print" || callName == "fmt.Printf" || callName == "fmt.Println") {
		return
	}

	if msg, banned := bannedLogPatterns[callName]; banned {
		reporter.Reportf(call.Pos(), "%s", msg)
	}
}

func checkWideEventContext(reporter *nolint.Reporter, info *logCallInfo) {
	// Check if the log has any of the required context fields
	hasContext := false
	for _, field := range info.fieldNames {
		fieldLower := strings.ToLower(field)
		for _, required := range requiredContextFields {
			if strings.Contains(fieldLower, required) || strings.Contains(required, fieldLower) {
				hasContext = true
				break
			}
		}
		// Also check for common alternatives
		if strings.Contains(fieldLower, "trace") ||
			strings.Contains(fieldLower, "span") ||
			strings.Contains(fieldLower, "request") ||
			strings.Contains(fieldLower, "req_id") ||
			strings.Contains(fieldLower, "correlation") {
			hasContext = true
		}
	}

	if !hasContext && len(info.fieldNames) > 0 {
		reporter.Reportf(info.call.Pos(),
			"wide event missing request context; add trace_id, request_id, or span_id for correlation")
	}
}

func getCallName(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		if ident, ok := fn.X.(*ast.Ident); ok {
			return ident.Name + "." + fn.Sel.Name
		}
	}
	return ""
}

// functionHasContext checks if the function has a context.Context parameter
func functionHasContext(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil {
		return false
	}

	for _, param := range fn.Type.Params.List {
		if isContextType(param.Type) {
			return true
		}
	}
	return false
}

// isContextType checks if an expression is context.Context
func isContextType(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == "context" && t.Sel.Name == "Context"
		}
	case *ast.Ident:
		// Could be aliased import or type alias
		return t.Name == "Context"
	}
	return false
}

// isSpanFromContextCall checks if a call is trace.SpanFromContext or similar
func isSpanFromContextCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	methodName := sel.Sel.Name

	// Check for trace.SpanFromContext, otel.SpanFromContext, etc.
	if spanFromContextFuncs[methodName] {
		// Check the package/receiver
		switch x := sel.X.(type) {
		case *ast.Ident:
			name := strings.ToLower(x.Name)
			if name == "trace" || name == "otel" || strings.Contains(name, "tracer") {
				return true
			}
		case *ast.SelectorExpr:
			// Could be oteltrace.SpanFromContext
			if x.Sel != nil {
				name := strings.ToLower(x.Sel.Name)
				if strings.Contains(name, "trace") {
					return true
				}
			}
		}
	}

	return false
}

// isSpanSetAttributesCall checks if a call is span.SetAttributes or similar
func isSpanSetAttributesCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	methodName := sel.Sel.Name

	// Check for SetAttributes, SetAttribute, AddEvent, etc.
	if spanSetAttributesMethods[methodName] {
		// Check if receiver looks like a span
		switch x := sel.X.(type) {
		case *ast.Ident:
			name := strings.ToLower(x.Name)
			if name == "span" || strings.Contains(name, "span") {
				return true
			}
		case *ast.CallExpr:
			// Could be trace.SpanFromContext(ctx).SetAttributes(...)
			if isSpanFromContextCall(x) {
				return true
			}
		}
	}

	return false
}
