package a

// Good: the file before this one ends with a function that takes a context;
// a package-level goroutine here still has no context to check
var _ = func() int {
	go func() {
		work()
	}()
	return 0
}()
