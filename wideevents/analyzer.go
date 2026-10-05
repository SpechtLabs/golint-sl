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
	"go/types"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"

	"github.com/spechtlabs/golint-sl/internal/nolint"
)

// Doc is the wideevents analyzer's documentation.
const Doc = `enforce wide event logging patterns instead of traditional logging

This analyzer implements the "logging sucks" philosophy (https://loggingsucks.com/):

1. BANS traditional loggers:
   - logrus.* (use zap instead)
   - log.* from stdlib (use zap instead)
   - fmt.Print/Printf/Println (use zap.Debug for dev output)

2. ENFORCES structured logging:
   - Require structured fields (zap.String, zap.Int, etc.; slog attributes
     or key-value pairs; fields added earlier in the chain with With or
     WithError)
   - Flag bare string messages without context
   - Suggest using span attributes for tracing

3. ENFORCES span attributes when context is available:
   - If a function has context.Context, use trace.SpanFromContext(ctx) to get the span
   - Add wide-event attributes to the span via span.SetAttributes()
   - Span attributes provide better observability than logs alone

4. DETECTS anti-patterns:
   - Multiple log statements in a single function (should be one wide event)
   - Info/Warn/Error logs without request context: a field whose name,
     ignoring case and the separators _ - and ., ends in trace_id, span_id,
     request_id, req_id, correlation_id, correlation, user_id, service or
     traceparent
   - Logging inside loops (creates log spam)
   - Functions with context that log but don't set span attributes

5. ALLOWS:
   - zap.Debug for development/troubleshooting
   - Single wide event emission at function end
   - Span attributes for OpenTelemetry integration
   - *Context methods and otelzap's Ctx(ctx) loggers without explicit
     request context fields, since they carry the trace context

Log calls are recognized by type: methods of the zap and otelzap loggers,
and functions and methods of log/slog, stdlib log and logrus. Other
functions named Info or Error (http.Error, status.Error) are not logging.

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

// Import paths of the logging packages the analyzer knows.
const (
	zapPath         = "go.uber.org/zap"
	zapcorePath     = "go.uber.org/zap/zapcore"
	slogPath        = "log/slog"
	stdlibLogPath   = "log"
	logrusPath      = "github.com/sirupsen/logrus"
	otelzapPath     = "github.com/uptrace/opentelemetry-go-extra/otelzap"
	spechtOtelzap   = "github.com/spechtlabs/go-otel-utils/otelzap"
	zapErrorFunc    = "Error"
	errorFieldName  = "error"
	printfSuffix    = "f"
	ctxLoggerSuffix = "WithCtx"
)

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

// bannedPackages maps the import path of each package with banned functions
// to the name bannedLogPatterns uses for it.
var bannedPackages = map[string]string{
	logrusPath:    "logrus",
	stdlibLogPath: "log",
	"fmt":         "fmt",
}

// loggerPackages maps the import path of each logging package to whether its
// package-level functions log (slog.Info, log.Fatal) or only the methods of
// its logger types do (zap's package-level functions build fields).
var loggerPackages = map[string]bool{
	zapPath:       false,
	otelzapPath:   false,
	spechtOtelzap: false,
	slogPath:      true,
	stdlibLogPath: true,
	logrusPath:    true,
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

// requestContextFields are the field names that correlate a wide event with
// a request, normalized by normalizeFieldName. A field counts when its
// normalized name ends in one of them (dd.trace_id, http.request_id).
var requestContextFields = []string{
	"traceid",
	"spanid",
	"requestid",
	"reqid",
	"correlationid",
	"correlation",
	"userid",
	"service",
	"traceparent",
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

// contextAwareMethods are methods that accept context.Context as first argument
// and automatically extract trace context (trace_id, span_id) from it
var contextAwareMethods = map[string]bool{
	"ErrorContext": true, // otelzap context-aware methods
	"InfoContext":  true,
	"WarnContext":  true,
	"DebugContext": true,
	"FatalContext": true,
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

		checkFunction(pass.TypesInfo, reporter, fn, isCLI)
	})

	return nil, nil
}

// funcScan collects what one function body does with loggers and spans.
type funcScan struct {
	info              *types.Info
	reporter          *nolint.Reporter
	logCalls          []*logCallInfo
	logsInLoops       []*ast.CallExpr
	isCLI             bool
	hasSpanUsage      bool
	hasSpanAttributes bool
}

// bodyVisitor walks a function body and tells the scan about every call,
// and whether the call sits inside a loop.
type bodyVisitor struct {
	scan   *funcScan
	inLoop bool
}

// Visit implements ast.Visitor.
func (v bodyVisitor) Visit(n ast.Node) ast.Visitor {
	switch node := n.(type) {
	case *ast.ForStmt, *ast.RangeStmt:
		return bodyVisitor{scan: v.scan, inLoop: true}
	case *ast.CallExpr:
		v.scan.visitCall(node, v.inLoop)
	}
	return v
}

// visitCall records a call: banned loggers are reported wherever they are,
// span usage counts wherever it is, and log calls inside loops are log spam
// rather than candidates for the wide event checks.
func (s *funcScan) visitCall(call *ast.CallExpr, inLoop bool) {
	checkBannedLogPatterns(s.reporter, s.info, call, s.isCLI)

	if isSpanFromContextCall(call) {
		s.hasSpanUsage = true
	}
	if isSpanSetAttributesCall(call) {
		s.hasSpanAttributes = true
	}

	info := analyzeLogCall(s.info, call)
	switch {
	case info == nil:
	case inLoop:
		s.logsInLoops = append(s.logsInLoops, call)
	default:
		s.logCalls = append(s.logCalls, info)
	}
}

func checkFunction(info *types.Info, reporter *nolint.Reporter, fn *ast.FuncDecl, isCLI bool) {
	scan := &funcScan{info: info, reporter: reporter, isCLI: isCLI}
	ast.Walk(bodyVisitor{scan: scan}, fn.Body)

	// Report logs inside loops
	for _, call := range scan.logsInLoops {
		reporter.Reportf(call.Pos(),
			"logging inside loop creates log spam; accumulate data and emit one wide event after the loop")
	}

	// Check for scattered log statements (multiple non-debug logs)
	nonDebugLogs := countNonDebugLogs(scan.logCalls)
	if nonDebugLogs > 1 {
		reporter.Reportf(fn.Pos(),
			"function has %d log statements; consider emitting a single wide event at the end instead of scattered logs",
			nonDebugLogs)
	}

	// Check each log call for required context
	for _, call := range scan.logCalls {
		if !call.isDebug && !call.hasStructuredFields {
			reporter.Reportf(call.call.Pos(),
				"log call without structured fields; use zap.String(\"field\", value) to add context for wide events")
		}

		// Check for traditional log methods that should be wide events
		if call.isTraditionalLog && !call.isDebug {
			checkWideEventContext(reporter, call)
		}
	}

	// If function has context and non-debug logs but doesn't use span
	// attributes, suggest it
	if functionHasContext(fn) && nonDebugLogs > 0 && !scan.hasSpanAttributes {
		if !scan.hasSpanUsage {
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
	hasContextMethod    bool // true if the logger carries the trace context (ErrorContext, otelzap Ctx(ctx))
}

// analyzeLogCall returns what the analyzer needs to know about call if it is
// a call to an Info/Warn/Error/Fatal/Debug method or function of a logger,
// and nil otherwise.
func analyzeLogCall(info *types.Info, call *ast.CallExpr) *logCallInfo {
	fn := loggerFunc(info, call)
	if fn == nil {
		return nil
	}

	method := fn.Name()
	if !traditionalLogMethods[method] && !allowedDebugMethods[method] {
		return nil
	}

	logCall := &logCallInfo{
		call:             call,
		method:           method,
		isDebug:          allowedDebugMethods[method],
		isTraditionalLog: traditionalLogMethods[method],
		hasContextMethod: contextAwareMethods[method] || isContextLogger(fn),
	}

	logCall.hasStructuredFields, logCall.fieldNames = logCallFields(info, fn, call)

	// Fields added earlier in the chain count too:
	// logger.With(zap.String("request_id", id)).Info("handled")
	if sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok {
		structured, names := chainFields(info, sel.X)
		logCall.hasStructuredFields = logCall.hasStructuredFields || structured
		logCall.fieldNames = append(logCall.fieldNames, names...)
	}

	return logCall
}

// loggerFunc returns the function or method call invokes if it belongs to a
// logging package: a method of one of its types, or a package-level function
// of a package whose package-level functions log. It returns nil otherwise.
func loggerFunc(info *types.Info, call *ast.CallExpr) *types.Func {
	fn, ok := typeutil.Callee(info, call).(*types.Func)
	if !ok || fn.Pkg() == nil {
		return nil
	}

	pkgFuncsLog, known := loggerPackages[fn.Pkg().Path()]
	if !known {
		return nil
	}
	if fn.Signature().Recv() == nil && !pkgFuncsLog {
		return nil
	}
	return fn
}

// isContextLogger reports whether fn is a method of an otelzap logger bound
// to a context (otelzap.L().Ctx(ctx).Info(...)), which adds the trace
// context to every event.
func isContextLogger(fn *types.Func) bool {
	recv := fn.Signature().Recv()
	if recv == nil {
		return false
	}

	t := recv.Type()
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}

	path := named.Obj().Pkg().Path()
	return (path == otelzapPath || path == spechtOtelzap) && strings.HasSuffix(named.Obj().Name(), ctxLoggerSuffix)
}

// logCallFields returns whether a log call passes structured fields and the
// names of those it can determine. Printf-style calls (Infof) and calls
// without a message parameter (log.Print, the sugared Info) print their
// arguments rather than attaching them as fields.
func logCallFields(info *types.Info, fn *types.Func, call *ast.CallExpr) (bool, []string) {
	if strings.HasSuffix(fn.Name(), printfSuffix) {
		return false, nil
	}

	params := fn.Signature().Params()
	if params.Len() < 2 || !isStringType(params.At(params.Len()-2).Type()) {
		return false, nil
	}

	return variadicFields(info, fn, call)
}

// chainFields returns the fields that calls earlier in a logger chain attach,
// e.g. the request_id in logger.With(zap.String("request_id", id)).Info(...).
// Named and WithOptions add no fields.
func chainFields(info *types.Info, recv ast.Expr) (bool, []string) {
	structured := false
	var names []string

	for {
		call, ok := ast.Unparen(recv).(*ast.CallExpr)
		if !ok {
			return structured, names
		}
		fn := loggerFunc(info, call)
		if fn == nil {
			return structured, names
		}

		s, n := chainCallFields(info, fn, call)
		structured = structured || s
		names = append(names, n...)

		sel, isSel := ast.Unparen(call.Fun).(*ast.SelectorExpr)
		if !isSel {
			return structured, names
		}
		recv = sel.X
	}
}

// chainCallFields returns the fields one call in a logger chain attaches.
func chainCallFields(info *types.Info, fn *types.Func, call *ast.CallExpr) (bool, []string) {
	switch fn.Name() {
	case "With":
		return variadicFields(info, fn, call)
	case "WithError":
		return true, []string{errorFieldName}
	case "WithField", "WithFields":
		if len(call.Args) == 0 {
			return true, nil
		}
		if key, ok := stringLiteral(call.Args[0]); ok {
			return true, []string{key}
		}
		return true, nil
	}
	return false, nil
}

// variadicFields reads the structured fields passed in the variadic
// parameter of a call to fn. A variadic parameter of field type (zap.Field,
// slog.Attr) holds one field per argument; one of type any holds key-value
// pairs, which may be mixed with fields (slog, the sugared *w methods).
func variadicFields(info *types.Info, fn *types.Func, call *ast.CallExpr) (bool, []string) {
	sig := fn.Signature()
	if !sig.Variadic() {
		return false, nil
	}

	last := sig.Params().Len() - 1
	if len(call.Args) <= last {
		return false, nil
	}
	args := call.Args[last:]

	slice, ok := sig.Params().At(last).Type().(*types.Slice)
	if !ok {
		return false, nil
	}
	elem := slice.Elem()
	fieldParam := isFieldType(elem)
	if !fieldParam && !types.IsInterface(elem) {
		return false, nil
	}

	// A spread slice (logger.Info(msg, fields...)) carries fields whose names
	// can't be read here.
	if call.Ellipsis.IsValid() {
		return true, nil
	}

	if fieldParam {
		return fieldArgs(info, args)
	}
	return keyValueArgs(info, args)
}

// fieldArgs reads arguments that are each one field.
func fieldArgs(info *types.Info, args []ast.Expr) (bool, []string) {
	var names []string
	for _, arg := range args {
		if name, ok := fieldName(info, arg); ok {
			names = append(names, name)
		}
	}
	return len(args) > 0, names
}

// keyValueArgs reads alternating keys and values, where a field (zap.Field,
// slog.Attr) may stand in for a key-value pair.
func keyValueArgs(info *types.Info, args []ast.Expr) (bool, []string) {
	structured := false
	var names []string

	for i := 0; i < len(args); i++ {
		t := info.TypeOf(args[i])
		switch {
		case t != nil && isFieldType(t):
			structured = true
			if name, ok := fieldName(info, args[i]); ok {
				names = append(names, name)
			}
		case t != nil && isStringType(t):
			structured = true
			if key, ok := stringLiteral(args[i]); ok {
				names = append(names, key)
			}
			i++ // skip the value
		}
	}
	return structured, names
}

// fieldName returns the key of a field built by a zap or slog constructor
// with a literal key (zap.String("request_id", id), slog.Int("status", 200));
// zap.Error(err) is the "error" field.
func fieldName(info *types.Info, arg ast.Expr) (string, bool) {
	call, ok := ast.Unparen(arg).(*ast.CallExpr)
	if !ok {
		return "", false
	}
	fn := typeutil.StaticCallee(info, call)
	if fn == nil || fn.Pkg() == nil || fn.Signature().Recv() != nil {
		return "", false
	}

	switch fn.Pkg().Path() {
	case zapPath:
		if fn.Name() == zapErrorFunc {
			return errorFieldName, true
		}
	case slogPath:
	default:
		return "", false
	}

	if len(call.Args) == 0 {
		return "", false
	}
	return stringLiteral(call.Args[0])
}

// isFieldType reports whether t is a structured logging field: zap.Field
// (zapcore.Field) or slog.Attr.
func isFieldType(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}

	switch named.Obj().Pkg().Path() {
	case zapPath, zapcorePath:
		return named.Obj().Name() == "Field"
	case slogPath:
		return named.Obj().Name() == "Attr"
	}
	return false
}

// isStringType reports whether t's underlying type is string.
func isStringType(t types.Type) bool {
	basic, ok := t.Underlying().(*types.Basic)
	return ok && basic.Kind() == types.String
}

// stringLiteral returns the value of expr if it is a string literal.
func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := ast.Unparen(expr).(*ast.BasicLit)
	if !ok {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	return value, err == nil
}

func checkBannedLogPatterns(reporter *nolint.Reporter, info *types.Info, call *ast.CallExpr, isCLI bool) {
	fn := typeutil.StaticCallee(info, call)
	if fn == nil || fn.Pkg() == nil || fn.Signature().Recv() != nil {
		return
	}
	pkgName, ok := bannedPackages[fn.Pkg().Path()]
	if !ok {
		return
	}
	callName := pkgName + "." + fn.Name()

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

	if len(info.fieldNames) > 0 && !slices.ContainsFunc(info.fieldNames, isRequestContextField) {
		reporter.Reportf(info.call.Pos(),
			"wide event missing request context; add trace_id, request_id, or span_id for correlation")
	}
}

// isRequestContextField reports whether a field name correlates the event
// with a request: its normalized name ends in one of requestContextFields.
// "id" or "user" alone do not.
func isRequestContextField(name string) bool {
	normalized := normalizeFieldName(name)
	for _, field := range requestContextFields {
		if strings.HasSuffix(normalized, field) {
			return true
		}
	}
	return false
}

// normalizeFieldName lower-cases name and drops the separators _ - and . so
// that trace_id, traceId, trace-id and dd.trace.id compare equal.
func normalizeFieldName(name string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '_', '-', '.':
			return -1
		}
		return unicode.ToLower(r)
	}, name)
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
