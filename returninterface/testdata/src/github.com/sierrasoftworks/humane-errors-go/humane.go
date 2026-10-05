// Package humane is a stub for testing the returninterface analyzer.
package humane

// Error represents a humane error with actionable advice.
type Error interface {
	error
	Advice() []string
}
