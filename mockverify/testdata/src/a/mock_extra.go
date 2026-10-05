package a

// Good: mock_*.go files are mock files; this one verifies MockOtherFile.
var _ Service = &MockOtherFile{}

// Bad: a mock in a mock_*.go file without verification.
type FakeExtra struct{} // want `mock "FakeExtra" should have compile-time interface verification`
