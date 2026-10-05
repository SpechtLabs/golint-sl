package a

import "testing"

// Good: test files are skipped.
func TestPanics(t *testing.T) {
	panic("test files may panic")
}
