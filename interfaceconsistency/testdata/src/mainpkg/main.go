package main

// ConcreteClient is a stub concrete type.
type ConcreteClient struct{}

func NewConcreteClient() *ConcreteClient {
	return &ConcreteClient{}
}

// Good: main packages are composition roots — creating concrete dependencies is expected.
func setupControllers() {
	_ = NewConcreteClient() // no diagnostic expected
}

func main() {
	setupControllers()
}
