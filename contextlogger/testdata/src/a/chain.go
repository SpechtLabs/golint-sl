package a

import (
	"context"

	"github.com/sirupsen/logrus"
	lr "github.com/sirupsen/logrus"
)

type notALogger struct{}

func (notALogger) Error(msg string) {}

// Bad: a chain on a global logger is one global logger call
func LogrusChain(ctx context.Context) {
	logrus.WithFields(logrus.Fields{}).Info("x") // want `function has context parameter but uses global logger`
}

// Bad: an aliased import of a logging package
func LogrusAliased(ctx context.Context) {
	lr.Info("x") // want `function has context parameter but uses global logger`
}

// Good: a variable whose name ends in "log" is not the log package
func Dialog(ctx context.Context) {
	var dialog notALogger
	dialog.Error("not a log")
}

// Good: a logger in a local variable is not a global logger
func LocalLogger(ctx context.Context) {
	logrus := lr.New()
	logrus.Info("x")
}
