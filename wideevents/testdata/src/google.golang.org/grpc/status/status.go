// Package status is a minimal stub of google.golang.org/grpc/status for the
// wideevents tests.
package status

// Error returns an error representing code and msg.
func Error(code uint32, msg string) error { return nil }

// Errorf returns an error representing code and a formatted message.
func Errorf(code uint32, format string, args ...any) error { return nil }
