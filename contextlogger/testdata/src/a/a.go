package a

import (
	"context"
	"log/slog"

	"example.com/log"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

type Service struct{}

// Good: logger taken from the context
func HandleGood(ctx context.Context) {
	logger := log.FromContext(ctx)
	logger.Info("handling request")
}

// Good: once FromContext is used, other global calls in the function are tolerated
func Mixed(ctx context.Context) {
	log.Info("start")
	log.FromContext(ctx).Info("done")
}

// Good: no context parameter, so the global logger is fine
func NoContext() {
	log.Info("no context here")
	logrus.Info("still fine")
}

// Bad: global structured logger while a context is available
func HandleBad(ctx context.Context, id string) {
	log.Info("handling", "id", id) // want `function has context parameter but uses global logger; use log.FromContext\(ctx\) instead`
	log.Error("failed", "id", id)  // want `function has context parameter but uses global logger`
	log.Warn("careful", "id", id)  // want `function has context parameter but uses global logger`
	log.Debug("details", "id", id) // want `function has context parameter but uses global logger`
}

// Bad: zap's global loggers
func ZapGlobal(ctx context.Context) {
	zap.L().Info("hello")          // want `function has context parameter but uses global logger`
	zap.S().Infow("hello", "k", 1) // want `function has context parameter but uses global logger`
}

// Bad: logrus package-level functions
func LogrusGlobal(ctx context.Context) {
	logrus.Info("a")              // want `function has context parameter but uses global logger`
	logrus.Error("b")             // want `function has context parameter but uses global logger`
	logrus.Warn("c")              // want `function has context parameter but uses global logger`
	logrus.Debug("d")             // want `function has context parameter but uses global logger`
	e := logrus.WithField("k", 1) // want `function has context parameter but uses global logger`
	e.Info("e")
}

// Good: calls that are not loggers, including ones the analyzer can only
// render with types.ExprString (index and literal callees)
func OtherCalls(ctx context.Context, fns []func()) {
	helper()
	fns[0]()
	func() {}()
}

func helper() {}

// Bad: a logger passed next to a context
func ZapParam(ctx context.Context, logger *zap.Logger) {} // want `logger passed as parameter alongside context; consider using log.FromContext\(ctx\) pattern instead`

func SugarParam(ctx context.Context, s *zap.SugaredLogger) {} // want `logger passed as parameter alongside context`

func LogrusParams(ctx context.Context, l *logrus.Logger, e *logrus.Entry) {} // want `logger passed as parameter alongside context` `logger passed as parameter alongside context`

func SlogParam(ctx context.Context, l *slog.Logger) {} // want `logger passed as parameter alongside context`

func OtelzapParam(ctx context.Context, l *otelzap.Logger) {} // want `logger passed as parameter alongside context`

// Good: a logger parameter without a context is not flagged
func LoggerOnly(logger *zap.Logger) {}

// Good: methods named init or main are skipped even with a context
func (s *Service) init(ctx context.Context) {
	log.Info("init")
}

func (s *Service) main(ctx context.Context) {
	log.Info("main")
}

// Good: nothing to check without a body
func external(ctx context.Context)

// Suppressed via nolint
func Suppressed(ctx context.Context) {
	log.Info("suppressed") //nolint:contextlogger
}
