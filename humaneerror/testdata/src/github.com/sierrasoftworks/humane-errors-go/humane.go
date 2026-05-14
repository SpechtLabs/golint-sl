// Package humane is a stub for testing the humaneerror analyzer.
package humane

// Error represents a humane error with actionable advice.
type Error interface {
	error
	Advice() []string
}

// Option configures a humane error (e.g. WithAdvice).
type Option interface {
	apply(Error)
}

// WithAdvice attaches one or more pieces of advice to a humane error.
func WithAdvice(advice ...string) Option {
	return nil
}

// New creates a new humane error with the given message and advice.
func New(message string, advice ...string) Error {
	return nil
}

// Newf creates a new humane error with a formatted message. Format args may
// intermix Option values (e.g. WithAdvice) with values consumed by the
// format string.
func Newf(format string, args ...any) Error {
	return nil
}

// Wrap wraps an existing error with a message and advice.
func Wrap(err error, message string, advice ...string) Error {
	return nil
}

// Wrapf wraps an existing error with a formatted message. Format args may
// intermix Option values (e.g. WithAdvice) with values consumed by the
// format string.
func Wrapf(err error, format string, args ...any) Error {
	return nil
}
