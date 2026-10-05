// Package mock allows the Mock prefix, the usual convention for mocks.
package mock

// Good: Mock* types in a mock package.
type MockClient struct{}

// Good: Mock* functions in a mock package.
func MockNew() *MockClient { return nil }
