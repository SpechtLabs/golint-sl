package a

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
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

// Good: interfaces are not pointers, whether declared here or imported.
func callIface(d Doer, s fmt.Stringer) string {
	d.Do()
	return s.String()
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

// Good: type parameters are not pointers, even when the constraint is an
// interface.
func generic[T fmt.Stringer](x T) string {
	return x.String()
}

// Good: a compound nil check with an early return covers every parameter
// in it.
func both(a, b *User) (string, error) {
	if a == nil || b == nil {
		return "", errNil
	}
	return a.Name + b.Name, nil
}

// Good: the use on the right of p != nil && or p == nil || is guarded, and
// so is the body of an if whose && condition includes p != nil.
func shortCircuit(u, v, w *User) bool {
	if u == nil || u.Age < 0 {
		return false
	}
	if v != nil && v.Age > 0 {
		println(v.Name)
	}
	return w != nil && w.Age > 0
}

// Good: !(p == nil) and the else of p == nil && ... rule nil out too.
func negated(u, v *User) {
	if !(u == nil) {
		println(u.Name)
	}
	if v == nil {
		return
	} else if v.Age > 0 {
		println(v.Name)
	}
}

// Good: a nil check whose body panics, exits, fails the test or breaks out
// of the loop leaves the parameter non-nil afterwards.
func terminating(a, b, c, d *User, tb testing.TB) {
	if a == nil {
		panic("nil")
	}
	if b == nil {
		os.Exit(1)
	}
	if c == nil {
		tb.Fatal("nil")
	}
	for range 3 {
		if d == nil {
			continue
		}
		println(d.Name)
	}
	println(a.Name, b.Name, c.Name)
}

// Good: a nil check that assigns a default leaves the parameter non-nil.
func defaulted(u *User) string {
	if u == nil {
		u = &User{}
	}
	return u.Name
}

// Bad: the use before the nil check is not covered by it.
func late(u *User) string {
	n := u.Name // want `pointer parameter "u" used without nil check`
	if u == nil {
		return ""
	}
	return n
}

// Bad: a nil check that only logs and falls through doesn't make the use
// after it safe.
func logOnly(u *User) string {
	if u == nil {
		println("nil")
	}
	return u.Name // want `pointer parameter "u" used without nil check`
}

// Bad: a use outside the guarded branch is not covered by it.
func outsideGuard(u *User) string {
	if u != nil && u.Age > 0 {
		println(u.Name)
	}
	return u.Name // want `pointer parameter "u" used without nil check`
}

// Bad: p == nil && ... being false doesn't rule nil out, nor does a nil
// check of another parameter.
func wrongCheck(u, v *User) {
	if u == nil && v == nil {
		return
	}
	println(u.Name) // want `pointer parameter "u" used without nil check`
}

// Good: a parameter shadowed by a local variable is a different variable.
func shadowed(u *User) {
	{
		u := &User{}
		println(u.Name)
	}
	if u == nil {
		return
	}
	println(u.Name)
}
