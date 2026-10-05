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
