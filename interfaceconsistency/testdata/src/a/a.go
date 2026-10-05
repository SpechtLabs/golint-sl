package a

import "net/http"

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

// Good: constructors may return the concrete type even when an interface
// for it exists ("accept interfaces, return structs").
func NewRepository() *Repository {
	return &Repository{}
}

// CacheInterface is the interface for Cache.
type CacheInterface interface {
	Lookup(key string) string
}

// Cache is the concrete implementation of CacheInterface.
type Cache struct{}

func (c *Cache) Lookup(key string) string { return key }

// Good: returning the interface is not reported either.
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

// --- One report per field and per call ---

// Bad: a name with two dependency words is still one dependency.
type Wiring struct {
	ServiceClient *ConcreteClient // want `field "ServiceClient" in struct "Wiring" looks like a dependency`
}

func (factory) NewStoreClient() *ConcreteClient { return &ConcreteClient{} }

// Bad: reported once although the name has two dependency words.
func wireStore(f factory) {
	_ = f.NewStoreClient() // want `creating NewStoreClient inside function`
}

// --- Fields of types from other packages, and type parameters ---

// Good: a pointer to another package's concrete type (*http.Client) is the
// normal way to hold it; only the package's own types are reported.
type Fetcher struct {
	HTTPClient *http.Client
}

// Good: a type parameter is not a concrete type.
type Pool[T any] struct {
	Client *T
}

// --- The idiomatic constructor shape ---

// Store is the interface consumers depend on.
type Store interface {
	Get()
}

type store struct{}

func (store) Get() {}

// Good: NewStore returns the concrete *store although the package declares
// a Store interface.
func NewStore() *store {
	return &store{}
}
