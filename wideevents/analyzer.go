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

// Doc is the wideevents analyzer's documentation.
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

// Analyzer reports traditional logging that should be wide events.
var Analyzer = &analysis.Analyzer{
	Name:     "wideevents",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Messages reported for banned logging calls.
const (
	logrusBannedMsg      = "logrus is banned; use zap with structured fields for wide events"
	logrusDebugBannedMsg = "logrus is banned; use zap.Debug with structured fields instead"
	stdlibLogBannedMsg   = "stdlib log is banned; use zap with structured fields for wide events"
	stdlibLogFatalMsg    = "stdlib log is banned; use zap.Fatal with structured fields instead"
	stdlibLogPanicMsg    = "stdlib log is banned; use zap.Panic with structured fields instead"
)

// zapErrorFunc is the zap field constructor for an error field.
const zapErrorFunc = "Error"

// Banned logging patterns - these should not be used
var bannedLogPatterns = map[string]string{
	// logrus - banned entirely
	"logrus.Info":       logrusBannedMsg,
	"logrus.Infof":      logrusBannedMsg,
	"logrus.Warn":       logrusBannedMsg,
	"logrus.Warnf":      logrusBannedMsg,
	"logrus.Error":      logrusBannedMsg,
	"logrus.Errorf":     logrusBannedMsg,
	"logrus.Fatal":      logrusBannedMsg,
	"logrus.Fatalf":     logrusBannedMsg,
	"logrus.Debug":      logrusDebugBannedMsg,
	"logrus.Debugf":     logrusDebugBannedMsg,
	"logrus.WithField":  logrusBannedMsg,
	"logrus.WithFields": logrusBannedMsg,

	// stdlib log - banned entirely
	"log.Print":   stdlibLogBannedMsg,
	"log.Printf":  stdlibLogBannedMsg,
	"log.Println": stdlibLogBannedMsg,
	"log.Fatal":   stdlibLogFatalMsg,
	"log.Fatalf":  stdlibLogFatalMsg,
	"log.Fatalln": stdlibLogFatalMsg,
	"log.Panic":   stdlibLogPanicMsg,
	"log.Panicf":  stdlibLogPanicMsg,
	"log.Panicln": stdlibLogPanicMsg,

	// fmt.Print - banned for logging (use for CLI output only)
	"fmt.Print":   "fmt.Print is not for logging; use zap.Debug for dev output or emit a wide event",
	"fmt.Printf":  "fmt.Printf is not for logging; use zap.Debug for dev output or emit a wide event",
	"fmt.Println": "fmt.Println is not for logging; use zap.Debug for dev output or emit a wide event",
}

// Traditional logging methods that should be replaced with wide events
var traditionalLogMethods = map[string]bool{
	"Info":         true,
	"Infof":        true,
	"Infow":        true,
	"InfoContext":  true, // otelzap context-aware methods
	"Warn":         true,
	"Warnf":        true,
	"Warnw":        true,
	"WarnContext":  true, // otelzap context-aware methods
	"Error":        true,
	"Errorf":       true,
	"Errorw":       true,
	"ErrorContext": true, // otelzap context-aware methods
	"Fatal":        true,
	"Fatalf":       true,
	"Fatalw":       true,
	"FatalContext": true, // otelzap context-aware methods
}

// Debug methods are allowed (for development)
var allowedDebugMethods = map[string]bool{
	"Debug":        true,
	"Debugf":       true,
	"Debugw":       true,
	"DebugContext": true, // otelzap context-aware methods
}

// Method chaining methods that add fields (otelzap/zap patterns)
var fieldChainMethods = map[string]bool{
	"WithError":   true, // otelzap.L().WithError(err)
	"With":        true, // logger.With(zap.String(...))
	"WithOptions": true, // logger.WithOptions(...)
	"Named":       true, // logger.Named("name") - adds logger name as context
}

// Required context fields for wide events
var requiredContextFields = []string{
	"request_id",
	"trace_id",
	"span_id",
	"dd.trace_id",
	"dd.span_id",
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

// isCLIPackage checks if the package path indicates CLI code where fmt.Print is acceptable
func isCLIPackage(pass *analysis.Pass) bool {
	pkgPath := pass.Pkg.Path()
	// Allow fmt.Print in cmd/ directories (CLI code) and internal/cli
	return strings.Contains(pkgPath, "/cmd/") ||
		strings.Contains(pkgPath, "/cli/") ||
		strings.HasSuffix(pkgPath, "/cmd") ||
		strings.HasSuffix(pkgPath, "/cli")
}

func run(pass *analysis.Pass) (any, error) {
	reporter := nolint.NewReporter(pass)
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	isCLI := isCLIPackage(pass)

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
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

		checkFunction(reporter, fn, isCLI)
	})

	return nil, nil
}

func checkFunction(reporter *nolint.Reporter, fn *ast.FuncDecl, isCLI bool) {
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
			logsInLoops = append(logsInLoops, logCallsInLoop(node)...)
			return false // Don't recurse again

		case *ast.CallExpr:
			// Check banned patterns first (skip fmt.Print* in CLI code)
			checkBannedLogPatterns(reporter, node, isCLI)

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
	nonDebugLogs := countNonDebugLogs(logCalls)
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

	// If function has context and non-debug logs but doesn't use span
	// attributes, suggest it
	if hasContext && nonDebugLogs > 0 && !hasSpanAttributes {
		if !hasSpanUsage {
			reporter.Reportf(fn.Pos(),
				"function has context.Context but doesn't use span attributes; "+
					"use span := trace.SpanFromContext(ctx) and span.SetAttributes() for better observability")
		} else {
			reporter.Reportf(fn.Pos(),
				"function gets span from context but doesn't set attributes; "+
					"add span.SetAttributes(attribute.String(\"key\", value)) for wide event data")
		}
	}
}

// logCallsInLoop returns the log calls anywhere inside the loop.
func logCallsInLoop(loop ast.Node) []*ast.CallExpr {
	var calls []*ast.CallExpr
	ast.Inspect(loop, func(inner ast.Node) bool {
		if call, ok := inner.(*ast.CallExpr); ok && analyzeLogCall(call) != nil {
			calls = append(calls, call)
		}
		return true
	})
	return calls
}

// countNonDebugLogs returns how many of the log calls are not debug logs.
func countNonDebugLogs(logCalls []*logCallInfo) int {
	nonDebugLogs := 0
	for _, info := range logCalls {
		if !info.isDebug {
			nonDebugLogs++
		}
	}
	return nonDebugLogs
}

type logCallInfo struct {
	call                *ast.CallExpr
	method              string
	fieldNames          []string
	isDebug             bool
	isTraditionalLog    bool
	hasStructuredFields bool
	hasContextMethod    bool // true if method is *Context (e.g., ErrorContext, InfoContext)
}

// contextAwareMethods are methods that accept context.Context as first argument
// and automatically extract trace context (trace_id, span_id) from it
var contextAwareMethods = map[string]bool{
	"ErrorContext": true, // otelzap context-aware methods
	"InfoContext":  true,
	"WarnContext":  true,
	"DebugContext": true,
	"FatalContext": true,
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

	// Only logger.Info(), logger.Error(), etc. are log calls, and only when the
	// receiver is not known to be something other than a logger
	if !traditionalLogMethods[method] && !allowedDebugMethods[method] {
		return nil
	}
	if isExcludedLogReceiver(sel.X) {
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
		hasContextMethod: contextAwareMethods[method],
	}

	// Check for structured fields in arguments
	info.hasStructuredFields, info.fieldNames = hasStructuredFields(call)

	// The receiver could be zap.L().Info(), otelzap.L().WithError(err).ErrorContext(), etc.
	// If method chaining adds fields, mark as having structured fields
	if x, ok := sel.X.(*ast.CallExpr); ok && hasFieldChaining(x) {
		info.hasStructuredFields = true
		info.fieldNames = append(info.fieldNames, "error") // WithError adds error field
	}

	return info
}

// isExcludedLogReceiver reports whether the receiver of a logger-named method
// is known not to be a logger (fmt, testing.T, errors, test assertion values).
func isExcludedLogReceiver(recv ast.Expr) bool {
	switch x := recv.(type) {
	case *ast.Ident:
		return isExcludedReceiverName(strings.ToLower(x.Name))
	case *ast.SelectorExpr:
		// Could be pkg.Logger or obj.logger, or struct.err.Error()
		if x.Sel == nil {
			return false
		}
		// Exclude struct fields that are errors (e.g., event.err.Error())
		fieldName := strings.ToLower(x.Sel.Name)
		return fieldName == "err" || strings.Contains(fieldName, "error")
	}
	return false
}

// isExcludedReceiverName reports whether a lower-cased receiver identifier
// names something other than a logger.
func isExcludedReceiverName(name string) bool {
	switch name {
	case "fmt", // fmt package - fmt.Errorf is not logging
		"t", "b", // testing.T methods - t.Errorf is not logging
		"got", "want", "expected", "actual": // common test assertion variables
		return true
	}
	// Exclude error variables - err.Error(), e.Error(), herr.Error(), lastCause.Error() is not logging
	return name == "e" || strings.Contains(name, "err") || strings.Contains(name, "cause")
}

// hasFieldChaining checks if a call expression has method chaining that adds fields
// e.g., otelzap.L().WithError(err) or logger.With(zap.String(...))
func hasFieldChaining(call *ast.CallExpr) bool {
	// Check if this call is a field-adding method
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		methodName := sel.Sel.Name
		if fieldChainMethods[methodName] {
			return true
		}
		// Recurse into the receiver to check for nested chaining
		if innerCall, ok := sel.X.(*ast.CallExpr); ok {
			return hasFieldChaining(innerCall)
		}
	}
	return false
}

func hasStructuredFields(call *ast.CallExpr) (bool, []string) {
	var fieldNames []string

	// Skip the first argument (message string)
	for i, arg := range call.Args {
		if i == 0 {
			continue // Skip message
		}

		// Check for zap.String(), zap.Int(), zap.Error(), etc.
		if name, ok := zapFieldName(arg); ok {
			fieldNames = append(fieldNames, name)
		}
	}

	return len(fieldNames) > 0, fieldNames
}

// zapFieldName returns the field name of a zap field constructor call such as
// zap.String("key", value), if it can be determined.
func zapFieldName(arg ast.Expr) (string, bool) {
	argCall, ok := arg.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	sel, ok := argCall.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok || ident.Name != "zap" {
		return "", false
	}

	// zap.Error() is a special case - the field name is "error"
	if sel.Sel.Name == zapErrorFunc || sel.Sel.Name == "NamedError" {
		return "error", true
	}

	// Extract field name if possible
	if len(argCall.Args) == 0 {
		return "", false
	}
	lit, ok := argCall.Args[0].(*ast.BasicLit)
	if !ok {
		return "", false
	}
	return strings.Trim(lit.Value, "\""), true
}

func checkBannedLogPatterns(reporter *nolint.Reporter, call *ast.CallExpr, isCLI bool) {
	callName := getCallName(call)
	if callName == "" {
		return
	}

	// Skip fmt.Print* in CLI code - it's used for user output, not logging
	if isCLI && (callName == "fmt.Print" || callName == "fmt.Printf" || callName == "fmt.Println") {
		return
	}

	if msg, banned := bannedLogPatterns[callName]; banned {
		reporter.Reportf(call.Pos(), "%s", msg)
	}
}

func checkWideEventContext(reporter *nolint.Reporter, info *logCallInfo) {
	// Context-aware methods (*Context like ErrorContext, InfoContext) automatically
	// extract trace context (trace_id, span_id) from the context parameter,
	// so they don't need explicit request context fields
	if info.hasContextMethod {
		return
	}

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
	if !spanFromContextFuncs[methodName] {
		return false
	}

	// Check the package/receiver
	switch x := sel.X.(type) {
	case *ast.Ident:
		name := strings.ToLower(x.Name)
		return name == "trace" || name == "otel" || strings.Contains(name, "tracer")
	case *ast.SelectorExpr:
		// Could be oteltrace.SpanFromContext
		return x.Sel != nil && strings.Contains(strings.ToLower(x.Sel.Name), "trace")
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
