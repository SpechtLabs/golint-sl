// The external test package's path ends in "_test", so it is exempt.
package ext_test

import "time"

func helperUsesTime() time.Time {
	return time.Now()
}
