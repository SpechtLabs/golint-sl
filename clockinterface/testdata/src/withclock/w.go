package withclock

import "time"

// Clock is the package's injectable time source.
type Clock interface {
	Now() time.Time
	After(d time.Duration) <-chan time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() } // want `direct time.Now\(\) call in business logic; use the Clock interface defined in this package`

// Bad: the package already has a Clock interface, so point at it
func Wait() {
	<-time.After(time.Second) // want `direct time.After\(\) call; use the Clock.After\(\) method instead`
}
