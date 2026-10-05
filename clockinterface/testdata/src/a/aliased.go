package a

import (
	. "time"
	t2 "time"
)

// Bad: an aliased import of time is still the time package
func AliasedNow() t2.Time {
	return t2.Now() // want `direct time.Now\(\) call in business logic`
}

// Bad: so is a dot import
func DotSleep() {
	Sleep(Second) // want `time.Sleep\(\) in business logic is usually a code smell`
}

// Good: time.Time.After is a method, not time.After
func Before(a, b t2.Time) bool {
	return b.After(a)
}

// Bad: names that only start with an exempt word are not exempt
func initializeCache() t2.Time {
	return t2.Now() // want `direct time.Now\(\) call in business logic`
}

func Newsletter() t2.Time {
	return t2.Now() // want `direct time.Now\(\) call in business logic`
}

func Stringify() string {
	return t2.Now().String() // want `direct time.Now\(\) call in business logic`
}

// Good: an exempt word followed by a new camel-case word
func NewCache() t2.Time {
	return t2.Now()
}

func Printf(format string) {
	_ = t2.Now()
}
