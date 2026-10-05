package a

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// Client is a local type sharing the name of http.Client.
type Client struct{ Name string }

type holder struct{ DefaultClient *http.Client }

type wrapper struct{ h holder }

type getter struct{}

func (getter) Get(u string) {}

func clients(t http.RoundTripper) []any {
	return []any{
		// Bad: http.Client literals without Timeout
		&http.Client{},                // want `http.Client without Timeout will wait forever; always set Timeout \(e.g., 30\*time.Second\)`
		http.Client{Transport: t},     // want `http.Client without Timeout`
		http.Client{nil, nil, nil, 0}, // want `http.Client without Timeout`

		// Good: Timeout is set
		&http.Client{Timeout: 30 * time.Second},
		&http.Client{Transport: t, Timeout: time.Second},
		[]*http.Client{{Timeout: time.Second}},

		// Good: other composite literals are ignored
		Client{Name: "local"},
		http.Header{},
		[]int{1, 2},
		struct{}{},
	}
}

func directCalls(ctx context.Context, g getter) {
	// Bad: package-level helpers that use DefaultClient or lack context
	_, _ = http.Get("https://example.com")                                // want `http.Get uses DefaultClient with no timeout; create a client with Timeout`
	_, _ = http.Post("https://example.com", "text/plain", nil)            // want `http.Post uses DefaultClient with no timeout; create a client with Timeout`
	_, _ = http.PostForm("https://example.com", url.Values{})             // want `http.PostForm uses DefaultClient with no timeout; create a client with Timeout`
	_, _ = http.Head("https://example.com")                               // want `http.Head uses DefaultClient with no timeout; create a client with Timeout`
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil) // want `http.NewRequest doesn't support context; use http.NewRequestWithContext instead`

	// Good: context-aware request construction and unrelated http functions
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com", nil)
	_ = http.StatusText(http.StatusOK)

	// Good: methods on other values and plain function calls
	g.Get("https://example.com")
	client := &http.Client{Timeout: time.Second}
	_, _ = client.Do(req)
	_ = clients(nil)

	// Bad: http.DefaultClient usage
	_, _ = http.DefaultClient.Do(req) // want `http.DefaultClient has no timeout and is shared globally; create your own http.Client with Timeout`
	c := http.DefaultClient           // want `http.DefaultClient has no timeout`
	_ = c

	// Good: DefaultClient fields on other values
	h := holder{DefaultClient: client}
	w := wrapper{h: h}
	_ = h.DefaultClient
	_ = w.h.DefaultClient
}

// Good: nolint suppresses the diagnostic
var shared = http.DefaultClient //nolint:httpclient

//nolint:golint-sl
var legacy = &http.Client{}

// Bad: nolint for another analyzer does not suppress
var other = &http.Client{} //nolint:nilcheck // want `http.Client without Timeout`
