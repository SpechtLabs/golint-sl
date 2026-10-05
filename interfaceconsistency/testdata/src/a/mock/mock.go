// Package mock holds mock implementations; mock bookkeeping skips it.
package mock

// MockClient is an exported interface in a mock package.
type MockClient interface {
	Do()
}

// FakeService is a mock service.
type FakeService struct{}

// Bad: dependency injection is still checked inside mock packages.
func wire() {
	_ = NewFakeService() // want `creating NewFakeService inside function`
}

// NewFakeService creates a FakeService.
func NewFakeService() *FakeService { return &FakeService{} }
