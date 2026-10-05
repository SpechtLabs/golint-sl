package a

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"

	"google.golang.org/grpc"
)

type holder struct{ f *os.File }

type cleaner struct{}

func (cleaner) Cleanup(fns ...func()) {}

func getResp() *http.Response   { return nil }
func wrap(f *os.File) *os.File  { return f }
func passthrough() *os.File     { return nil }
func closeIt(f *os.File) func() { return func() { _ = f.Close() } }

// Good: deferred close of the response body.
func deferBody() {
	resp, err := http.Get("https://example.com")
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// Good: deferred, plain, and assigned closes.
func closes() {
	a, _ := os.Open("a")
	defer a.Close()

	b, _ := os.Open("b")
	b.Close()

	c, _ := os.Open("c")
	_ = c.Close()
}

// Good: close inside a deferred function literal.
func deferFuncLit() {
	f, _ := os.Create("x")
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()
}

// Good: close checked in an if-init.
func closeInIfInit() error {
	f, err := os.Create("x")
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
}

// Good: t.Cleanup with a close inside the function literal.
func cleanup(t *testing.T) {
	f, _ := os.Open("x")
	t.Cleanup(func() {
		_ = f.Close()
	})
}

// Good: closes inside an if body and an else block (expression, assignment
// and defer forms).
func closesInBranches(ok bool) {
	a, _ := os.Open("a")
	b, _ := os.Open("b")
	c, _ := os.Open("c")
	d, _ := os.Open("d")
	if ok {
		a.Close()
		defer d.Close()
	} else if !ok {
		b.Close()
	} else {
		_ = c.Close()
		defer d.Close()
	}
}

// Good: the resource is created in an if-init and closed in the body.
func ifInitCreateAndClose(ch chan int) {
	if f, err := os.Create("x"); err == nil {
		<-ch
		x, y := 1, 2
		use(x, y)
		fmt.Println(f.Name())
		_ = f.Close()
	}
	if g, err := os.Create("y"); err == nil {
		g.Close()
	}
}

// Good: created in an if-init, closed with a defer in the else block.
func ifInitElseDefer() error {
	if f, err := os.Open("x"); err != nil {
		return err
	} else {
		defer f.Close()
	}
	return nil
}

// Good: gRPC connection and response body closed through selectors.
func grpcClosed() {
	cc, err := grpc.Dial("localhost:443")
	if err != nil {
		return
	}
	defer cc.Close()
}

// Good: os.Stdout and friends must not be closed.
func stdio() {
	out := os.Stdout
	errOut, in := os.Stderr, os.Stdin
	use(out, errOut, in)
}

// Good: values that do not come from a create function are not tracked,
// nor are struct fields, plain copies, or calls through other expressions.
func untracked(h *holder) error {
	f := passthrough()
	g := f
	var err error
	h.f, err = os.Open("x")
	n := func() int { return 1 }()
	field := h.f
	other := (&holder{}).f
	args := os.Args
	use(f, g, n, field, other, args)
	return err
}

// Good: shapes that are not closes are ignored.
func notCloses(c cleaner, t *testing.T) {
	f, _ := os.Open("x")
	use()
	fmt.Println("x")
	c.Cleanup()
	t.Cleanup(closeIt(f))
	wrap(f).Close()
	getResp().Body.Close()
	if fmt.Sprint(); true {
		x := 0
		x++
	}
	var i int
	if i++; i > 0 {
		_ = f.Close()
	}
}

// Good: suppressed with a nolint directive.
func suppressed() {
	f, _ := os.Open("x") //nolint:resourceclose
	use(f)
}

// Good: a function declared without a body (implemented elsewhere) is skipped.
func external() *os.File

type fileParser struct{ f *os.File }

// Good: the close is the return value.
func closeInReturn(data []byte) error {
	f, err := os.Create("out")
	if err != nil {
		return err
	}
	_, _ = f.Write(data)
	return f.Close()
}

// Good: returning the resource hands it to the caller.
func openConfig() (*os.File, error) {
	f, err := os.Open("config")
	if err != nil {
		return nil, err
	}
	return f, nil
}

// Good: returning a struct literal that holds the resource hands it to the
// caller.
func newFileParser(path string) (*fileParser, error) {
	f, err := os.Open(path)
	return &fileParser{f: f}, err
}

// Good: returning the response body hands it to the caller.
func fetch(url string) (any, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	return (resp.Body), nil
}

// Good: a resource assigned to a named result is returned.
func openNamed() (f *os.File, err error) {
	f, err = os.Open("x")
	return
}

// Good: a typed dial whose connection is closed.
func dialTCPClosed(addr *net.TCPAddr) error {
	c, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}
	defer c.Close()
	return nil
}

// Good: a close inside a goroutine's function literal.
func closeInGoroutine(done chan struct{}) {
	f, _ := os.Open("x")
	go func() {
		defer f.Close()
		<-done
	}()
}
