package a

// Good: a method without a body (implemented in assembly) on a type with a
// mutex has nothing to check, and must not crash the analyzer.
func (c *Counter) External() int
