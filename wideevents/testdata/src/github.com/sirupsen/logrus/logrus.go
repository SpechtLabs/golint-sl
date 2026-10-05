// Package logrus is a minimal stub of github.com/sirupsen/logrus for the
// wideevents tests.
package logrus

// Entry is a log entry with fields.
type Entry struct{}

func Info(args ...any)                       {}
func Infof(format string, args ...any)       {}
func Warn(args ...any)                       {}
func WithField(key string, value any) *Entry { return &Entry{} }
