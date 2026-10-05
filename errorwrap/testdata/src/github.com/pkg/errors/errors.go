// Package errors is a stub of github.com/pkg/errors for testing the errorwrap
// analyzer.
package errors

// Wrap annotates err with a message.
func Wrap(err error, message string) error { return err }

// Wrapf annotates err with a formatted message.
func Wrapf(err error, format string, args ...any) error { return err }

// WithMessage annotates err with a message without a stack trace.
func WithMessage(err error, message string) error { return err }
