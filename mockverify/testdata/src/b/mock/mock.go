// Package mock lives in a mock/ directory, so every file in it is a mock file.
package mock

type Clock interface{ Now() int }

// Good: verified.
type FakeClock struct{}

func (*FakeClock) Now() int { return 0 }

var _ Clock = &FakeClock{}

// Bad: not verified.
type StubClock struct{} // want `mock "StubClock" should have compile-time interface verification: var _ InterfaceName = &StubClock\{\}`
