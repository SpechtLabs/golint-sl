// Package log is a stub of a structured logging package with a global logger
// and context helpers.
package log

import "context"

type Logger struct{}

func (l *Logger) Info(msg string, kv ...any) {}

func FromContext(ctx context.Context) *Logger { return &Logger{} }

func IntoContext(ctx context.Context, l *Logger) context.Context { return ctx }

func Info(msg string, kv ...any)  {}
func Error(msg string, kv ...any) {}
func Warn(msg string, kv ...any)  {}
func Debug(msg string, kv ...any) {}
