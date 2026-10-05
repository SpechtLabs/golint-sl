// Package otelzap is a minimal stub of
// github.com/uptrace/opentelemetry-go-extra/otelzap for the wideevents tests.
package otelzap

import (
	"context"

	"go.uber.org/zap"
)

// Logger is a context-aware zap logger.
type Logger struct{}

// L returns the global Logger.
func L() *Logger { return &Logger{} }

func (l *Logger) WithError(err error) *Logger                                  { return l }
func (l *Logger) ErrorContext(ctx context.Context, msg string, f ...zap.Field) {}
func (l *Logger) InfoContext(ctx context.Context, msg string, f ...zap.Field)  {}
