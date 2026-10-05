// Package zap is a minimal stub of go.uber.org/zap.
package zap

type Field struct{}

type Logger struct{}

func (l *Logger) Info(msg string, fields ...Field) {}

func String(key, value string) Field { return Field{} }

func Any(key string, value any) Field { return Field{} }
