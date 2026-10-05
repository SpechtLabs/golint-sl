package ext

import "time"

// Bad: the in-package code is still checked
func Stamp() time.Time {
	return time.Now() // want `direct time.Now\(\) call`
}
