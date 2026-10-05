// Package a holds the interfaces the mocks implement. This file is not a
// mock file, so it declares no mock-named structs.
package a

import "fmt"

type Service interface{ Do() }

type Plain struct{}

func (Plain) Do() {}

func (*MockVerified) Do()    {}
func (*FakePtrVerified) Do() {}
func (*MockUnverified) Do()  {}
func (*StubSuppressed) Do()  {}
func (*MockOtherFile) Do()   {}
func (*MockBlankMulti) Do()  {}
func (*MockNamedVar) Do()    {}

// Good: the composite-literal verification for a mock in a_mock.go.
var _ Service = &MockVerified{}

// Good: the typed-nil verification for a mock in a_mock.go.
var _ Service = (*FakePtrVerified)(nil)

// Not verifications: none of these mark a mock as verified.
var (
	_, _             = 1, 2                       // more than one name
	namedVar Service = &MockNamedVar{}            // name is not the blank identifier
	_        Service                              // no value
	_, _     Service = &MockBlankMulti{}, Plain{} // two names and two values
	_                = -1                         // unary operator other than &
	plainVal         = Plain{}
	_                = &plainVal     // & of a non-composite
	_                = &struct{}{}   // composite type is not an identifier
	_        Service = &Plain{}      // not a mock name
	_                = fmt.Sprint()  // call whose Fun is not parenthesised
	_                = (int)(1)      // parenthesised Fun that is not a pointer
	_                = (*[]int)(nil) // pointer to a non-identifier
	_        Service = (*Plain)(nil) // pointer to a non-mock name
	_                = 1             // basic literal
)
