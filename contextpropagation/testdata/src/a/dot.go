package a

import . "context"

// Bad: a dot-imported Context is recognised as a context parameter
func DotImported(c Context) {} // want `context parameter "c" is received but never used`
