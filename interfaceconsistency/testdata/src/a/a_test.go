package a

// Good: test files may construct concrete dependencies inline.
func setupFixture() {
	_ = NewConcreteClient()
}

// TestDouble is an exported interface declared in a test file.
type TestDouble interface {
	Do()
}
