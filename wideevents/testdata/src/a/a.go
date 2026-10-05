package a

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// --- Banned loggers ---

// Bad: stdlib log is banned.
func useStdlibLog() {
	log.Println("starting")    // want `stdlib log is banned; use zap with structured fields for wide events`
	log.Printf("count %d", 42) // want `stdlib log is banned; use zap with structured fields for wide events`
}

// Bad: log.Fatal is banned and, being a Fatal* method on a "log" receiver,
// is also a log call without structured fields.
func useStdlibFatal() {
	log.Fatalf("boom %d", 1) // want `stdlib log is banned; use zap.Fatal with structured fields instead` `log call without structured fields`
}

// Bad: logrus is banned.
func useLogrus() {
	logrus.WithField("key", "value") // want `logrus is banned; use zap with structured fields for wide events`
}

// Bad: fmt.Print* is not for logging outside CLI packages.
func useFmtPrint() {
	fmt.Println("debug output") // want `fmt.Println is not for logging; use zap.Debug for dev output or emit a wide event`
}

// Good: fmt.Errorf and fmt.Sprintf build values, they do not log.
func buildValues(name string) error {
	_ = fmt.Sprintf("hello %s", name)
	return fmt.Errorf("bad %s", name)
}

// --- Structured logging ---

// Good: a single wide event carrying request context.
func wideEvent(logger *zap.Logger, id string) {
	logger.Info("request handled", zap.String("request_id", id), zap.Int("status", 200))
}

// Bad: a bare message without structured fields.
func messageOnly(logger *zap.Logger) {
	logger.Warn("something happened") // want `log call without structured fields; use zap.String`
}

// Bad: the sugared logger is still a log call without structured fields.
func sugared(sugar *zap.SugaredLogger) {
	sugar.Infof("count %d", 3) // want `log call without structured fields`
}

// Bad: structured fields, but none that correlate the event with a request.
func missingContext(logger *zap.Logger, err error) {
	logger.Error("failed", zap.Error(err)) // want `wide event missing request context; add trace_id, request_id, or span_id`
}

// Bad: a named error field is still not request context.
func missingContextNamedError(logger *zap.Logger, err error) {
	logger.Error("failed", zap.NamedError("cause", err)) // want `wide event missing request context`
}

// Good: a correlation field counts as request context.
func correlated(logger *zap.Logger, id string) {
	logger.Info("done", zap.String("correlation", id))
}

// Good: a non-literal field key yields no field name, but the literal
// trace_id field still provides request context.
func dynamicKey(logger *zap.Logger, key, id string) {
	logger.Info("done", zap.Any(key, id), zap.String("trace_id", id), describe(id))
}

// Bad: several non-debug log statements in one function.
func scattered(logger *zap.Logger, id string) { // want `function has 2 log statements; consider emitting a single wide event`
	logger.Info("step one", zap.String("trace_id", id))
	logger.Info("step two", zap.String("span_id", id))
}

// Good: debug logs are allowed and are not counted as scattered logs.
func debugOnly(logger *zap.Logger) {
	logger.Debug("verbose detail")
	logger.Debug("more detail")
}

// Good: the global zap.L() logger with request context.
func globalLogger(id string) {
	zap.L().Info("global", zap.String("trace_id", id))
}

// Good: logger fields reached through a struct field.
type server struct {
	logger *zap.Logger
	tracer trace.Tracer
}

func (s *server) serve(id string) {
	s.logger.Info("served", zap.String("request_id", id))
}

// --- Logging in loops ---

// Bad: logging inside a range loop.
func logInRangeLoop(logger *zap.Logger, items []string) {
	for _, it := range items {
		logger.Info("item", zap.String("request_id", it)) // want `logging inside loop creates log spam`
	}
}

// Bad: even debug logs inside a for loop are log spam.
func logInForLoop(logger *zap.Logger) {
	for i := 0; i < 3; i++ {
		logger.Debug("tick", zap.Int("i", i)) // want `logging inside loop creates log spam`
	}
}

// --- Span attributes when a context is available ---

// Bad: a context is available but no span is used.
func ctxNoSpan(ctx context.Context, logger *zap.Logger, id string) { // want `function has context.Context but doesn't use span attributes`
	logger.Info("handled", zap.String("request_id", id))
}

// Bad: the span is fetched but no attributes are set.
func ctxSpanNoAttrs(ctx context.Context, logger *zap.Logger, id string) { // want `function gets span from context but doesn't set attributes`
	span := trace.SpanFromContext(ctx)
	defer span.End()
	logger.Info("handled", zap.String("request_id", id))
}

// Bad: a span started from a tracer struct field, without attributes.
func (s *server) handle(ctx context.Context, id string) { // want `function gets span from context but doesn't set attributes`
	_, span := s.tracer.Start(ctx, "handle")
	defer span.End()
	s.logger.Info("handled", zap.String("request_id", id))
}

// Good: span attributes are set alongside the log.
func ctxWithSpanAttrs(ctx context.Context, logger *zap.Logger, id string) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes("request_id", id)
	logger.Info("handled", zap.String("request_id", id))
}

// Good: an event added directly on the span from the context.
func ctxChainedSpan(ctx context.Context, logger *zap.Logger, id string) {
	trace.SpanFromContext(ctx).AddEvent("handled")
	logger.Info("handled", zap.String("request_id", id))
}

// Good: a span started from a tracer parameter with its status set.
func ctxTracerParam(ctx context.Context, tracer trace.Tracer, logger *zap.Logger, id string) {
	_, span := tracer.Start(ctx, "op")
	span.SetStatus(1, "ok")
	logger.Info("done", zap.String("request_id", id))
}

// Good: a context with only debug logs needs no span attributes.
func ctxDebugOnly(ctx context.Context, logger *zap.Logger) {
	logger.Debug("detail", zap.String("k", "v"))
}

// Context is a local alias for context.Context.
type Context = context.Context

// Bad: an aliased context type still counts as a context parameter.
func aliasCtx(ctx Context, logger *zap.Logger, id string) { // want `function has context.Context but doesn't use span attributes`
	logger.Info("handled", zap.String("request_id", id))
}

// Good: otelzap context-aware methods carry the trace context and WithError
// adds the error field.
func otelzapContext(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes("op", "x")
	otelzap.L().WithError(err).ErrorContext(ctx, "failed")
}

// Good: debug through a chained logger is allowed.
func chainedDebug(logger *zap.Logger) {
	logger.Named("worker").Sugar().Debugw("chained")
	newLogger().Debug("from a constructor call")
}

// --- Calls that are not logging ---

type event struct {
	err error
}

// Good: testing.T, error values and assertion variables are not loggers.
func notLogging(t *testing.T, err, got, v error, ev *event) string {
	t.Errorf("unexpected %v", err)
	_ = err.Error()
	_ = got.Error()
	_ = v.Error()
	return ev.err.Error()
}

// Good: an immediately invoked function literal and a plain call.
func plainCalls() {
	func() {}()
	_ = newLogger()
}

// --- Skipped functions ---

// Good: init, main and Test* functions are skipped.
func init() {
	log.Println("init")
}

func main() {
	log.Println("main")
}

func TestLikeHelper() {
	log.Println("test helper")
}

// Good: functions without a body are skipped.
func noBody()

// --- nolint suppression ---

// Good: diagnostics silenced by nolint directives; other analyzers' names
// do not silence wideevents.
func suppressed() {
	log.Println("inline") //nolint:wideevents

	//nolint:golint-sl
	fmt.Println("preceding line")

	log.Println("other analyzer") //nolint:contextfirst // want `stdlib log is banned`
}

func newLogger() *zap.Logger { return zap.L() }

func describe(id string) zap.Field { return zap.String("id", id) }
