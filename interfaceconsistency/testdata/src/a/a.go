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

// --- More struct field checks ---

// Good: a value (non-pointer) field is not reported.
type ValueWiring struct {
	Provider ConcreteClient
}

// Good: field names that do not look like dependencies are ignored.
type PlainConfig struct {
	Timeout *int
	Name    string
}

// --- Constructors with a matching interface ---

// IRepository is the interface for Repository.
type IRepository interface {
	Get() string
}

// Repository is the concrete implementation of IRepository.
type Repository struct{}

func (r *Repository) Get() string { return "" }

// Bad: an IRepository interface exists but the constructor returns *Repository.
func NewRepository() *Repository { // want `constructor "NewRepository" returns concrete type; consider returning interface "IRepository"`
	return &Repository{}
}

// CacheInterface is the interface for Cache.
type CacheInterface interface {
	Lookup(key string) string
}

// Cache is the concrete implementation of CacheInterface.
type Cache struct{}

func (c *Cache) Lookup(key string) string { return key }

// Good: the constructor returns the matching CacheInterface.
func NewCache() CacheInterface {
	return &Cache{}
}

// Good: constructors without results are not checked.
func NewNothing() {}

// --- More dependency injection checks ---

type factory struct{}

func (factory) NewClientFor(name string) *ConcreteClient { return &ConcreteClient{} }

func NewTestClient() *ConcreteClient { return &ConcreteClient{} }

// Bad: creating a client through a method call.
func wireThroughFactory(f factory) {
	_ = f.NewClientFor("x") // want `creating NewClientFor inside function`
}

// Good: NewTest* helpers and New* calls that are not dependencies are fine,
// as are calls through function values.
func wireHelpers(mk func() int) {
	_ = NewTestClient()
	_ = NewNothingSpecial()
	_ = mk()
	_ = func() int { return 0 }()
}

func NewNothingSpecial() int { return 0 }

// Good: functions without a body are skipped.
func linkedElsewhere()

// --- nolint suppression ---

// Good: diagnostics silenced by nolint directives.
type SuppressedWiring struct {
	Client *ConcreteClient //nolint:interfaceconsistency

	//nolint:golint-sl
	Store *ConcreteClient

	Resolver *ConcreteClient //nolint:optionspattern // want `field "Resolver" in struct "SuppressedWiring" looks like a dependency`
}

// unexportedIface is not a candidate for a mock.
type unexportedIface interface {
	run()
}
