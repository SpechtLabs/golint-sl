package a

// MyClient is a stub interface for testing.
type MyClient interface {
	Do()
}

// ConcreteClient is a stub concrete type.
type ConcreteClient struct{}

func (c *ConcreteClient) Do() {}

// --- Struct field checks ---

// Bad: field looks like a dependency but uses concrete type
type BadService struct {
	Client *ConcreteClient // want `field "Client" in struct "BadService" looks like a dependency`
}

// Good: field uses an interface
type GoodService struct {
	Client MyClient
}

// Good: field has a json tag — this is a DTO/CRD field, not a dependency
type APISpec struct {
	ServiceSelector *ConcreteClient `json:"serviceSelector,omitempty"`
}

// Good: field has a json tag alongside other tags
type MixedTagSpec struct {
	MyMiddleware *ConcreteClient `json:"middleware" yaml:"middleware"`
}

// Bad: field name matches pattern and has NO json tag
type InternalWiring struct {
	Middleware *ConcreteClient // want `field "Middleware" in struct "InternalWiring" looks like a dependency`
}

// --- Constructor return type checks ---

// Good: no matching interface, so no report
func NewConcreteClient() *ConcreteClient {
	return &ConcreteClient{}
}

// --- Dependency injection checks ---

// Bad: creating a NewClient inside a non-constructor function
func SetupRoutes() {
	_ = NewConcreteClient() // want `creating NewConcreteClient inside function`
}

// Good: constructor functions are allowed to create things
func NewMyService() *BadService {
	c := NewConcreteClient()
	return &BadService{Client: c}
}
