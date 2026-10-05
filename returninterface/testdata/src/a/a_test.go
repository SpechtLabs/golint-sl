package a

// Good: test files may return interfaces.
func fakeStorage() Storage { return &fileStorage{} }
