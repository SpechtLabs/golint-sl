package a

import (
	"errors"
	"log"
	"net/http"
	"testing"
	"time"

	v1 "example.com/api/v1"
)

var errNil = errors.New("nil")

type User struct {
	Name string
	Age  int
}

type Doer interface{ Do() }

// Context is a local interface named like context.Context.
type Context interface{ Value() }

type UserPtr *User

// Good: the documented pattern, nil check with early return first.
func ProcessUser(user *User) error {
	if user == nil {
		return errNil
	}
	println(user.Name)
	return nil
}

// Good: reversed operands and != with an early return count as checks.
func reversedCheck(u *User, v *User) error {
	if nil == u {
		return errNil
	}
	if v != nil {
		println(v.Name)
	}
	return errors.New(u.Name)
}

// Good: an empty nil-check body with the use in the else branch.
func emptyCheckBody(u *User) {
	if u == nil {
	} else {
		println(u.Name)
	}
}

// Bad: a field access without a nil check, reported once per parameter.
func useField(u *User) int {
	println(u.Name) // want `pointer parameter "u" used without nil check; add 'if u == nil \{ return ... \}' at function start`
	return u.Age
}

// Bad: an explicit dereference without a nil check.
func deref(n *int) int {
	return *n // want `pointer parameter "n" dereferenced without nil check; add 'if n == nil \{ return ... \}' at function start`
}

// Bad: indexing a pointer to an array without a nil check.
func index(a *[3]int) int {
	return a[0] // want `pointer parameter "a" indexed without nil check`
}

// Bad: conditions that are not nil checks do not count.
func notNilChecks(u *User, ok bool, other *User) {
	if ok {
		return
	}
	if u == other {
		return
	}
	if u.Age > 0 { // want `pointer parameter "u" used without nil check`
		return
	}
}

// Bad: a named pointer type is a pointer.
func namedPointer(p UserPtr) string {
	return p.Name // want `pointer parameter "p" used without nil check`
}

// Bad: a local interface value used without a nil check.
func callIface(d Doer) {
	d.Do() // want `pointer parameter "d" used without nil check`
}

// Good: error and Context interfaces, and interface literals, are skipped.
func trustedIfaces(e error, cx Context, v interface{ M() }) string {
	cx.Value()
	v.M()
	return e.Error()
}

// Good: non-pointer parameters are not checked.
func values(n int, s string, tm time.Time) int64 {
	return tm.Unix() + int64(n+len(s))
}

// Good: trusted parameter names are skipped.
func trustedNames(req *User, cfg *User, opts *User) string {
	return req.Name + cfg.Name + opts.Name
}

// Good: trusted framework types are skipped, by exact match, by the
// pointer form of a value selector type, and by API version pattern.
func trustedTypes(tt *testing.T, lg *log.Logger, rq http.Request, th *v1.Thing) {
	tt.Log(lg.Prefix(), rq.Method, th.Name)
}

// Good: receivers are not parameters.
func (u *User) Describe() string {
	return u.Name
}

// Good: a function without parameters.
func noParams() {}

// Good: suppressed by nolint.
func suppressed(u *User) string {
	return u.Name //nolint:nilcheck
}
