package info // want `^mocks=\[FakeB MockA StubC\] verified=\[FakeB MockA\] unverified=\[StubC\]$`

type I interface{}

type MockA struct{}

type FakeB struct{}

type StubC struct{}

// Not a struct, so not a mock.
type MockI interface{}

// Not a mock name.
type Plain struct{}

var _ I = &MockA{}

var _ I = (*FakeB)(nil)
