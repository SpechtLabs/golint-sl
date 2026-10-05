package a

import "time"

// Clock here is a struct, not an interface, so it does not count as the
// package's Clock interface and the generic suggestion is used.
type Clock struct{}

type Service struct {
	cache cache
}

type cache struct{ t timer }

type timer struct{}

func (timer) Now() time.Time { return time.Time{} }

// Bad: direct time.Now() in business logic
func Expired(deadline time.Time) bool {
	return time.Now().After(deadline) // want `direct time.Now\(\) call in business logic; inject a Clock interface for testability`
}

// Bad: direct time.After()
func WaitForIt() {
	<-time.After(time.Second) // want `direct time.After\(\) call; inject a Clock interface with After\(\) method`
}

// Bad: time.Sleep()
func Backoff() {
	time.Sleep(time.Second) // want `time.Sleep\(\) in business logic is usually a code smell`
}

// Bad: tickers and timers
func Poll() {
	t := time.NewTicker(time.Second) // want `direct time.NewTicker\(\) call; consider abstracting time operations for testability`
	defer t.Stop()
	tm := time.NewTimer(time.Second) // want `direct time.NewTimer\(\) call; consider abstracting time operations for testability`
	defer tm.Stop()
}

// Bad: method without a clock parameter; calls inside closures count too
func (s *Service) Touch() {
	f := func() time.Time {
		return time.Now() // want `direct time.Now\(\) call in business logic`
	}
	_ = f
}

// Good: other time functions and conversions are not flagged
func Durations(sec int) time.Duration {
	d := time.Duration(sec) * time.Second
	_ = time.Unix(0, 0)
	return d
}

// Good: calls that are not time.X selectors are ignored
func NotTime(s *Service) time.Time {
	helper()
	return s.cache.t.Now()
}

func helper() {}

// Good: the function receives a Clock, so it is assumed to use it
func WithClock(clk Clock) time.Time {
	return time.Now()
}

// Good: exempt function names (constructors, formatting, printing)
func NewService() *Service {
	_ = time.Now()
	return &Service{}
}

func FormatAge(t time.Time) string {
	return time.Now().Sub(t).String()
}

func PrintUptime() {
	_ = time.Now()
}

func (s *Service) String() string {
	return time.Now().String()
}

func init() {
	_ = time.Now()
}

func main() {
	_ = time.Now()
}

// Good: a function declaration without a body has nothing to check
func linkedElsewhere()

// Suppressed via nolint
func Suppressed() time.Time {
	return time.Now() //nolint:clockinterface
}
