// Package nofuncs has no function declarations, so its package-level closure
// has no enclosing function to capture from.
package nofuncs

var x1, x2, x3, x4, x5, x6 int

// Bad: still checked for nesting, without a capture check
var Handler = func(a bool) int { // want `closure has nesting depth of 3`
	if a {
		if a {
			if a {
				return x1 + x2 + x3 + x4 + x5 + x6
			}
		}
	}
	return 0
}
