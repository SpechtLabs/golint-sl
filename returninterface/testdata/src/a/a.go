// Package a holds the returninterface cases.
package a

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"

	humane "github.com/sierrasoftworks/humane-errors-go"
)

type Storage interface {
	Save(data []byte) error
}

type ValidationError interface {
	error
	Field() string
}

type fileStorage struct{}

func (*fileStorage) Save([]byte) error { return nil }

// Good: no results.
func Process(Storage) {}

// Good: concrete return type.
func Load() *fileStorage { return &fileStorage{} }

// Bad: returns a project interface.
func Storage1() Storage { // want `function "Storage1" returns interface "Storage"; return concrete type instead`
	return &fileStorage{}
}

// Bad: each interface result is reported, concrete ones are not.
func Pair() (Storage, int, error) { // want `function "Pair" returns interface "Storage"`
	return nil, 0, nil
}

// Bad: an inline interface literal.
func Inline() interface{ Save([]byte) error } { // want `function "Inline" returns interface "interface\{Save\(\[\]byte\) error\}"`
	return nil
}

// Bad: a third-party interface that is not on the allow list.
func Mux() http.ResponseWriter { // want `function "Mux" returns interface "http.ResponseWriter"`
	return nil
}

// Good: factory prefixes may return interfaces, case-insensitively.
func NewStorage() Storage     { return &fileStorage{} }
func CreateStorage() Storage  { return &fileStorage{} }
func BuildStorage() Storage   { return &fileStorage{} }
func MakeStorage() Storage    { return &fileStorage{} }
func GetStorage() Storage     { return &fileStorage{} }
func OpenStorage() Storage    { return &fileStorage{} }
func ConnectStorage() Storage { return &fileStorage{} }
func newStorage() Storage     { return &fileStorage{} }

// Good: methods are skipped (they may implement interfaces).
func (*fileStorage) Clone() Storage { return &fileStorage{} }

// Good: the empty interface is left to the emptyinterface analyzer.
func Anything() any          { return nil }
func Something() interface{} { return nil }

// Good: allow-listed standard interfaces.
func Err() error                   { return nil }
func Humane() humane.Error         { return nil }
func Reader() io.Reader            { return nil }
func Writer() io.Writer            { return nil }
func Closer() io.Closer            { return nil }
func ReadCloser() io.ReadCloser    { return nil }
func ReadWriter() io.ReadWriter    { return nil }
func Ctx() context.Context         { return context.Background() }
func Str() fmt.Stringer            { return nil }
func Sorter() sort.Interface       { return nil }
func Handler() http.Handler        { return nil }
func Transport() http.RoundTripper { return nil }

// Good: interfaces whose name ends in "error" are idiomatic error types.
func Validate() ValidationError { return nil }

// Good: constraint-typed generics whose constraint is "any".
func Identity[T any](v T) T { return v }

// Good: suppressed with a nolint directive.
func Suppressed() Storage { //nolint:returninterface
	return nil
}
