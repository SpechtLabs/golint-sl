// Package logrus is a minimal stub of github.com/sirupsen/logrus for the
// wideevents tests.
package logrus

// Entry is a log entry with fields.
type Entry struct{}

func Info(args ...any)                       {}
func Infof(format string, args ...any)       {}
func Warn(args ...any)                       {}
func WithField(key string, value any) *Entry { return &Entry{} }

// Fields is a set of log fields.
type Fields map[string]any

func WithFields(fields Fields) *Entry { return &Entry{} }

func (e *Entry) Info(args ...any) {}
func (e *Entry) Warn(args ...any) {}
