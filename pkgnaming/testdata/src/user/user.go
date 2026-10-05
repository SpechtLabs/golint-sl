// Package user is a good name; it checks stutter in exported names.
package user

// Good: the type is exactly the package name.
type User struct{}

// Good: the remainder after the package name starts lower case.
type Username string

// Bad: user.UserService stutters.
type UserService struct{} // want `type user.UserService stutters; consider renaming to user.Service`

// Bad: an exported function stutters.
func UserByID(id string) *User { // want `function user.UserByID stutters; consider renaming to user.ByID`
	return nil
}

// Good: the name does not start with the package name.
func NewUser() *User { return &User{} }

// Good: methods are not checked.
func (User) UserName() string { return "" }

// Good: unexported functions are not checked.
func userLookup() {}

// Good: suppressed by nolint.
type UserStore struct{} //nolint:pkgnaming

// Good: callers never write user.userState, so an unexported type can't
// stutter.
type userState struct{}

// Good: a function-local type is invisible outside the function, exported
// name or not.
func localTypes() {
	type userLocal struct{}
	type UserLocal struct{}
	_ = userLocal{}
	_ = UserLocal{}
	_ = userState{}
}
