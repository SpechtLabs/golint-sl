package pattern

import clock "time"

// An aliased import still counts as a direct call, and a method named After
// on time.Time does not.
func aliased(t clock.Time) bool {
	return clock.Now().After(t)
}
