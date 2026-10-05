// Package leak has a package-level closure ahead of a function whose locals
// share the names of the package variables the closure uses.
package leak

var a1, a2, a3, a4, a5, a6 int

// Good: package-level variables are not captured from an enclosing function
var Handler = func() int { return a1 + a2 + a3 + a4 + a5 + a6 }

func later() {
	a1, a2, a3, a4, a5, a6 := 1, 2, 3, 4, 5, 6
	println(a1, a2, a3, a4, a5, a6)
}
