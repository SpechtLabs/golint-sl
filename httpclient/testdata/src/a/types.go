package a

import (
	"net/http"
	nethttp "net/http"
	"time"
)

// ClientAlias is a type alias of http.Client.
type ClientAlias = http.Client

// Bad: a literal of an alias of http.Client without Timeout
var aliased = ClientAlias{} // want `http.Client without Timeout will wait forever`

// Good: an aliased literal with Timeout
var aliasedTimeout = ClientAlias{Timeout: time.Second}

// Bad: a renamed net/http import is still net/http
func renamed(u string) {
	_, _ = nethttp.Get(u)                    // want `http.Get uses DefaultClient with no timeout; create a client with Timeout`
	_ = nethttp.DefaultClient                // want `http.DefaultClient has no timeout and is shared globally`
	_ = &nethttp.Client{}                    // want `http.Client without Timeout will wait forever`
	_, _ = nethttp.NewRequest("GET", u, nil) // want `http.NewRequest doesn't support context`
}

type fake struct{ DefaultClient int }

func (fake) Get(string) {}

// Good: a variable named http is not the net/http package
func shadowed() {
	http := fake{}
	http.Get("x")
	_ = http.DefaultClient
}

// Good: an unkeyed literal with a non-zero Timeout
var unkeyed = http.Client{nil, nil, nil, time.Second}

// Bad: a constant zero Timeout, keyed or unkeyed, is no timeout
var (
	zeroKeyed   = http.Client{Timeout: 0}                     // want `http.Client without Timeout will wait forever`
	zeroUnkeyed = http.Client{nil, nil, nil, 0 * time.Second} // want `http.Client without Timeout will wait forever`
)

// Good: a Timeout set from a variable
func fromConfig(d time.Duration) *http.Client { return &http.Client{Timeout: d} }

// Bad: an element literal with an elided type is an http.Client too
var pool = []*http.Client{{}} // want `http.Client without Timeout will wait forever`

var _ = []any{aliased, aliasedTimeout, unkeyed, zeroKeyed, zeroUnkeyed, pool}
