// Package main checks the main() exception.
package main

import "errors"

// Good: one-off errors in main() are accepted.
func main() {
	_ = errors.New("startup failed")
}

// Good: a package-level sentinel declared after main() is accepted because
// it is package level, not because it follows main().
var ErrAfterMain = errors.New("after main")

// Bad: other functions in package main are still checked.
func run() error {
	return errors.New("run failed") // want `inline errors.New\(\) in function "run"`
}
