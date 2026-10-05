// Package main checks the main() exception.
package main

import "errors"

// Good: one-off errors in main() are accepted.
func main() {
	_ = errors.New("startup failed")
}

// Bad: other functions in package main are still checked.
func run() error {
	return errors.New("run failed") // want `inline errors.New\(\) in function "run"`
}
