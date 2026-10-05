// Package zap is a minimal stub of go.uber.org/zap.
package zap

type Logger struct{}

type SugaredLogger struct{}

func (l *Logger) Info(msg string)                    {}
func (s *SugaredLogger) Infow(msg string, kv ...any) {}

func L() *Logger        { return &Logger{} }
func S() *SugaredLogger { return &SugaredLogger{} }
