package plain

import "time"

// Clock is a struct here, not an interface, so it does not count.
type Clock struct{}

type MockClockFactory struct{}

func use(c Clock) {
	_ = time.Now()
	_ = c
}
