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

// LoggerWithCtx is a Logger bound to a context.
type LoggerWithCtx struct{}

// Ctx binds the Logger to ctx.
func (l *Logger) Ctx(ctx context.Context) LoggerWithCtx { return LoggerWithCtx{} }

func (l LoggerWithCtx) Info(msg string, f ...zap.Field) {}
