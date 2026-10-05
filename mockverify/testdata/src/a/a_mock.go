package a

// Good: verified with var _ Service = &MockVerified{} in a.go.
type MockVerified struct{}

// Good: verified with var _ Service = (*FakePtrVerified)(nil) in a.go.
type FakePtrVerified struct{}

// Bad: no verification anywhere in the package.
type MockUnverified struct{} // want `mock "MockUnverified" should have compile-time interface verification: var _ InterfaceName = &MockUnverified\{\}`

// Bad: only assigned to a named variable, which is not the verification pattern.
type MockNamedVar struct{} // want `mock "MockNamedVar" should have compile-time interface verification`

// Bad: a two-name blank declaration is not the verification pattern.
type MockBlankMulti struct{} // want `mock "MockBlankMulti" should have compile-time interface verification`

// Good: suppressed by nolint.
type StubSuppressed struct{} //nolint:mockverify

// Good: non-struct mock-named types are not checked.
type MockIface interface{ Do() }

type MockFunc func()

// Good: the verification can live in another file of the package.
type MockOtherFile struct{}

// Good: Stub, Mock and Fake are only mock names as whole words, so neither
// Stubborn, Mockingbird nor Fakeable is a mock.
type StubbornWorker2 struct{}

type Mockingbird struct{}

type Fakeable struct{}

// Bad: Fake as the trailing word is a mock name too.
type StoreFake struct{} // want `mock "StoreFake" should have compile-time interface verification`

// Good: verified with var _ Service = MockValue{} in a.go.
type MockValue struct{}

// Good: verified with var _ Service = new(MockNew) in a.go.
type MockNew struct{}

// Good: verified with var _ Service = &MockGeneric[int]{} in a.go.
type MockGeneric[T any] struct{ v T }

// Good: verified with var _ Service = (*FakeGenericPtr[string])(nil) in a.go.
type FakeGenericPtr[T any] struct{ v T }
