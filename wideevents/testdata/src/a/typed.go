package a

import (
	"context"
	stdfmt "fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
)

// --- Only real loggers are logging ---

// Good: http.Error writes a response, it does not log.
func notFound(w http.ResponseWriter) {
	http.Error(w, "not found", http.StatusNotFound)
}

// Good: status.Error builds a gRPC error, it does not log.
func grpcErrors() error {
	_ = status.Errorf(5, "missing %s", "x")
	return status.Error(5, "not found")
}

type reporter struct{}

func (reporter) Info(msg string)  {}
func (reporter) Error(msg string) {}

// Good: a type of this package with Info and Error methods is not a logger.
func localReporter(r reporter) {
	r.Info("started")
	r.Error("failed")
}

type printer struct{}

func (printer) Println(args ...any) {}

// Good: a local value called fmt is not the fmt package.
func shadowedFmt() {
	fmt := printer{}
	fmt.Println("not fmt")
}

// Bad: a renamed fmt import is still fmt.
func renamedFmt() {
	stdfmt.Println("renamed") // want `fmt.Println is not for logging`
}

// --- Fields added earlier in the chain ---

// Good: the request_id field comes from With.
func chainedWith(logger *zap.Logger, id string) {
	logger.With(zap.String("request_id", id)).Info("handled")
}

// Good: Named adds no field, but the call's own request_id does.
func namedWithField(logger *zap.Logger, id string) {
	logger.Named("api").Info("handled", zap.String("request_id", id))
}

// Bad: Named adds a logger name, not a field.
func namedOnly(logger *zap.Logger) {
	logger.Named("api").Info("handled") // want `log call without structured fields`
}

// Bad: With adds a field, but not one that correlates the event.
func chainedWithoutContext(ctx context.Context, logger *zap.Logger) { // want `function has context.Context but doesn't use span attributes`
	logger.With(zap.String("component", "api")).Info("handled") // want `wide event missing request context`
}

// Good: a logger bound to a context carries the trace context.
func boundToContext(ctx context.Context, id string) {
	trace.SpanFromContext(ctx).SetAttributes("id", id)
	otelzap.L().Ctx(ctx).Info("handled", zap.String("component", "api"))
}

// --- Banned calls and span usage inside loops ---

// Bad: banned calls are reported inside loops too.
func bannedInLoop(items []string) {
	for _, it := range items {
		stdfmt.Printf("%s\n", it) // want `fmt.Printf is not for logging`
		log.Println(it)           // want `stdlib log is banned`
	}
}

// Good: span attributes set inside a loop count.
func spanInLoop(ctx context.Context, logger *zap.Logger, items []string) {
	span := trace.SpanFromContext(ctx)
	for _, it := range items {
		span.SetAttributes("item", it)
	}
	logger.Info("done", zap.String("request_id", "x"))
}

// Bad: nested loops report a log call once.
func nestedLoops(logger *zap.Logger, rows [][]string) {
	for _, row := range rows {
		for _, cell := range row {
			logger.Debug("cell", zap.String("cell", cell)) // want `logging inside loop creates log spam`
		}
	}
}

// --- Request context fields ---

// Bad: "id" is not request context, although request_id contains it.
func idOnly(ctx context.Context, logger *zap.Logger, id string) { // want `function has context.Context but doesn't use span attributes`
	logger.Info("done", zap.String("id", id)) // want `wide event missing request context`
}

// Bad: neither is "user".
func userOnly(ctx context.Context, logger *zap.Logger, user string) { // want `function has context.Context but doesn't use span attributes`
	logger.Info("done", zap.String("user", user)) // want `wide event missing request context`
}

// Good: prefixed and camel-cased spellings of the context fields count.
func datadogTraceID(logger *zap.Logger, id string) {
	logger.Info("done", zap.String("dd.trace_id", id))
}

func prefixedRequestID(logger *zap.Logger, id string) {
	logger.Info("done", zap.String("http.request_id", id))
}

func camelCaseTraceID(logger *zap.Logger, id string) {
	logger.Info("done", zap.String("traceId", id))
}

// --- log/slog and the sugared logger ---

// Good: slog key-value pairs and attributes are structured fields.
func slogFields(id string) {
	slog.Info("handled", "request_id", id)
}

// Good: an slog attribute with request context.
func slogAttr(id string) {
	slog.Info("handled", slog.String("trace_id", id))
}

// Bad: an slog call without any fields.
func slogBare() {
	slog.Info("handled") // want `log call without structured fields`
}

// Bad: slog fields without request context.
func slogNoContext(ctx context.Context) { // want `function has context.Context but doesn't use span attributes`
	slog.Info("handled", "count", 1) // want `wide event missing request context`
}

// Good: methods of an slog.Logger, with fields from With.
func slogLogger(l *slog.Logger, id string) {
	l.With("request_id", id).Warn("slow")
}

// Good: the sugared logger's *w methods take key-value pairs.
func sugaredKeyValues(sugar *zap.SugaredLogger, id string) {
	sugar.Infow("handled", "request_id", id)
}

// Good: fields spread from a slice are structured, their names unknown.
func spreadFields(logger *zap.Logger, fields []zap.Field) {
	logger.Info("handled", fields...)
}

// Good: a field held in a variable is a structured field; its name is
// unknown, so request context is not checked.
func fieldVariable(logger *zap.Logger, field zap.Field) {
	logger.Info("handled", field)
}

// Bad: log.Fatal prints its arguments; it has no message to attach fields to.
func stdlibFatal(err error) {
	log.Fatal("boom", err) // want `stdlib log is banned; use zap.Fatal` `log call without structured fields`
}

// Bad: logrus is banned, but a field from WithField still counts for the
// wide event checks (three log calls are scattered logs, too).
func logrusChain(id, key string) { // want `function has 3 log statements`
	logrus.WithField("request_id", id).Info("handled")              // want `logrus is banned`
	logrus.WithField(key, id).Warn("dynamic key")                   // want `logrus is banned`
	logrus.WithFields(logrus.Fields{"request_id": id}).Info("many") // want `logrus is banned`
}

// Good: without a context there is no request to correlate with, as in
// startup, shutdown and CLI code, so fields without request context are fine.
func shutdownEvent(logger *zap.Logger) {
	logger.Info("observability shutdown", zap.String("trace_flush", "ok"))
}

// Good: the same holds for a closure that a function without a context returns.
func shutdownFunc(logger *zap.Logger) func() {
	return func() {
		logger.Info("observability shutdown", zap.String("phase", "flush"))
	}
}
