// Package a holds the resourceclose cases: bad.go has the leaks, good.go
// the correctly closed resources and the skipped shapes.
package a

import (
	"database/sql"
	"net"
	"net/http"
	"os"

	"google.golang.org/grpc"
)

func use(...any) {}

// Open is a project wrapper that hands out a file.
func Open(name string) (*os.File, error) { return nil, nil } //nolint:resourceclose

// Bad: http.Get response body is never closed.
func httpGet() {
	resp, err := http.Get("https://example.com") // want `HTTP response body must be closed: defer resp\.Body\.Close\(\)`
	use(resp, err)
}

// Bad: closing the response itself does not count; the Body must be closed.
func clientDo(c *http.Client, req *http.Request) {
	resp, err := c.Do(req) // want `HTTP response body must be closed`
	use(resp, err)
}

// Bad: os.Open, os.Create, os.OpenFile and os.CreateTemp.
func files() {
	a, _ := os.Open("a")                         // want `file must be closed: defer f\.Close\(\)`
	b, _ := os.Create("b")                       // want `file must be closed`
	c, _ := os.OpenFile("c", os.O_RDONLY, 0o600) // want `file must be closed`
	d, _ := os.CreateTemp("", "d")               // want `file must be closed`
	use(a, b, c, d)
}

// Bad: a project function whose name matches a create function.
func wrapper() {
	f, err := Open("x") // want `file must be closed`
	use(f, err)
}

// Bad: plain assignment to an existing variable.
func reassigned() {
	var f *os.File
	var err error
	f, err = os.Open("x") // want `file must be closed`
	use(f, err)
}

// Bad: database rows and prepared statements.
func database(db *sql.DB) {
	rows, _ := db.Query("SELECT 1")   // want `database rows must be closed: defer rows\.Close\(\)`
	stmt, _ := db.Prepare("SELECT 1") // want `prepared statement must be closed: defer stmt\.Close\(\)`
	use(rows, stmt)
}

// Bad: network and gRPC connections.
func connections() {
	conn, _ := net.Dial("tcp", "localhost:80") // want `connection must be closed: defer conn\.Close\(\)`
	cc, _ := grpc.NewClient("localhost:443")   // want `gRPC connection must be closed: defer conn\.Close\(\)`
	use(conn, cc)
}

// Bad: created in an if-init but the body only uses it.
func ifInitNoClose(other *os.File) {
	if f, err := os.Open("x"); err == nil { // want `file must be closed`
		use(f)
		_ = other.Close()
	}
}

// Bad: closing a different variable inside the if-init body.
func ifInitClosesOther(other *os.File) {
	if f, err := os.Open("x"); err == nil { // want `file must be closed`
		other.Close()
		use(f)
	}
}
