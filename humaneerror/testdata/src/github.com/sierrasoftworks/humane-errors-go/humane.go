// Package humane is a stub for testing the humaneerror analyzer.
package humane

// Error represents a humane error with actionable advice.
type Error interface {
	error
	Advice() []string
}

// New creates a new humane error with the given message and advice.
func New(message string, advice ...string) Error {
	return nil
}

// Wrap wraps an existing error with a message and advice.
func Wrap(err error, message string, advice ...string) Error {
	return nil
}
