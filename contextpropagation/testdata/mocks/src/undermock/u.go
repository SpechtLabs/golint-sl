// Package undermock lives in a checkout whose absolute path contains a
// mocks/ directory, which says nothing about the package itself.
package undermock

import "context"

// Bad: still checked
func Get(ctx context.Context) {} // want `context parameter "ctx" is received but never used`
