// Package zap is a minimal stub of go.uber.org/zap.
package zap

type Logger struct{}

func L() *Logger                      { return &Logger{} }
func S() *SugaredLogger               { return &SugaredLogger{} }
func (*Logger) Fatal(msg string)      {}
func (*Logger) Panic(msg string)      {}
func (*Logger) DPanic(msg string)     {}
func (*Logger) Info(msg string)       {}
func (*Logger) Sugar() *SugaredLogger { return &SugaredLogger{} }

type SugaredLogger struct{}

func (*SugaredLogger) Fatalw(msg string, kv ...any) {}
func (*SugaredLogger) Panicw(msg string, kv ...any) {}
func (*SugaredLogger) Infow(msg string, kv ...any)  {}
