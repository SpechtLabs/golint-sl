package a

import (
	stderrors "errors"
	stdfmt "fmt"
)

// Bad: the calls are resolved through the type checker, so renamed imports
// of errors and fmt are still the standard library functions.
func renamedImports() {
	_ = stderrors.New("renamed") // want `avoid errors.New\(\); use humane.New`
	_ = stdfmt.Errorf("renamed") // want `avoid fmt.Errorf\(\); use humane.Wrap`
}

type formatter struct{}

func (formatter) Errorf(format string, args ...any) string { return format }

type factory struct{}

func (factory) New(text string) string { return text }

// Good: methods named Errorf and New on values that merely happen to be
// called fmt and errors are not the standard library functions.
func shadowedNames() {
	fmt := formatter{}
	errors := factory{}
	_ = fmt.Errorf("not the fmt package")
	_ = errors.New("not the errors package")
}
