// Package zap is a minimal stub of go.uber.org/zap for the wideevents tests.
package zap

// Field is a structured logging field.
type Field struct{}

// Option configures a Logger.
type Option interface{}

// Logger is a structured logger.
type Logger struct{}

// SugaredLogger is the printf-style logger.
type SugaredLogger struct{}

// L returns the global Logger.
func L() *Logger { return &Logger{} }

// S returns the global SugaredLogger.
func S() *SugaredLogger { return &SugaredLogger{} }

// String constructs a string field.
func String(key, val string) Field { return Field{} }

// Int constructs an int field.
func Int(key string, val int) Field { return Field{} }

// Any constructs a field of any type.
func Any(key string, val any) Field { return Field{} }

// Error constructs an "error" field.
func Error(err error) Field { return Field{} }

// NamedError constructs a named error field.
func NamedError(key string, err error) Field { return Field{} }

func (l *Logger) Debug(msg string, fields ...Field) {}
func (l *Logger) Info(msg string, fields ...Field)  {}
func (l *Logger) Warn(msg string, fields ...Field)  {}
func (l *Logger) Error(msg string, fields ...Field) {}
func (l *Logger) Fatal(msg string, fields ...Field) {}

func (l *Logger) With(fields ...Field) *Logger       { return l }
func (l *Logger) Named(name string) *Logger          { return l }
func (l *Logger) WithOptions(opts ...Option) *Logger { return l }
func (l *Logger) Sugar() *SugaredLogger              { return &SugaredLogger{} }
func (l *Logger) Sync() error                        { return nil }

func (s *SugaredLogger) Debugw(msg string, kv ...any)   {}
func (s *SugaredLogger) Infof(tmpl string, args ...any) {}
func (s *SugaredLogger) Infow(msg string, kv ...any)    {}
func (s *SugaredLogger) Errorw(msg string, kv ...any)   {}
