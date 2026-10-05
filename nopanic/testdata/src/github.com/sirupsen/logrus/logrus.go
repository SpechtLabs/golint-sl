// Package logrus is a minimal stub of github.com/sirupsen/logrus.
package logrus

func Fatal(args ...any)                 {}
func Fatalf(format string, args ...any) {}
func Fatalln(args ...any)               {}
func Panic(args ...any)                 {}
func Panicf(format string, args ...any) {}
func Panicln(args ...any)               {}
func Info(args ...any)                  {}

type Logger struct{}

func New() *Logger                         { return &Logger{} }
func (*Logger) Fatal(args ...any)          {}
func (*Logger) Panicf(f string, a ...any)  {}
func (*Logger) Info(args ...any)           {}
func (*Logger) WithError(err error) *Entry { return &Entry{} }

type Entry struct{}

func WithError(err error) *Entry         { return &Entry{} }
func (*Entry) Fatal(args ...any)         {}
func (*Entry) Fatalf(f string, a ...any) {}
func (*Entry) Panicln(args ...any)       {}
func (*Entry) Error(args ...any)         {}
