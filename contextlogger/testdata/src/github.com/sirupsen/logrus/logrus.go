// Package logrus is a minimal stub of github.com/sirupsen/logrus.
package logrus

type Fields map[string]any

type Logger struct{}

type Entry struct{}

func (e *Entry) Info(args ...any) {}

func Info(args ...any)  {}
func Error(args ...any) {}
func Warn(args ...any)  {}
func Debug(args ...any) {}

func WithField(key string, value any) *Entry { return &Entry{} }
func WithFields(fields Fields) *Entry        { return &Entry{} }

func New() *Logger { return &Logger{} }

func (l *Logger) Info(args ...any) {}
