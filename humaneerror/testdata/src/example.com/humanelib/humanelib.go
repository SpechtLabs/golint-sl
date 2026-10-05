// Package humanelib is a stub humane-style error package imported under a
// custom alias, exercising the analyzer's import-path fallback.
package humanelib

// Error is an error with advice.
type Error interface {
	error
}

// New creates an error with advice.
func New(message string, advice ...string) Error { return nil }

// Wrap wraps an error with advice.
func Wrap(err error, message string, advice ...string) Error { return nil }
