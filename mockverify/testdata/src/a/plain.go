package a

// Good: plain.go is not a mock file, so mock-named structs in it are not
// checked.
type MockInPlainFile struct{}

// Good: not a mock file, and Stubborn is not the word Stub either.
type StubbornWorker struct{}
