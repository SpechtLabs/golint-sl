package a

import "time"

// Good: test files in the package itself are exempt like external test packages
func fixtureTime() time.Time {
	return time.Now()
}
