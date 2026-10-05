// Package store sits under a directory called main, which is not an exemption.
package store

import "time"

// Bad: a main/ path segment does not exempt the package
func Touch() time.Time {
	return time.Now() // want `direct time.Now\(\) call in business logic`
}
