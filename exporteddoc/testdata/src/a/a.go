// Package a exercises the exporteddoc analyzer.
//
// Case annotations (Good:/Bad:) are separated from the declarations by a blank
// line so that they do not become doc comments themselves.
package a

import "errors"

// Good: documented exported function starting with its name

// Documented does something useful.
func Documented() {}

// Bad: exported function without documentation

func Undocumented() {} // want `exported function Undocumented should have a documentation comment`

// Bad: documentation that does not start with the function name

// handles requests // want `documentation for Misnamed should start with "Misnamed"`
func Misnamed() {}

// Good: unexported functions need no documentation

func helper() {}

// Server is a documented type.
type Server struct{}

// Good: methods are skipped even when undocumented

func (s *Server) Start() {}

// Good: documented type with a doc starting with its name

// Service handles business logic.
type Service struct{}

// Bad: exported type without documentation

type Bare struct{} // want `exported type Bare should have a documentation comment`

// Bad: type documentation not starting with the type name

// holds configuration // want `documentation for Config should start with "Config"`
type Config struct{}

// Good: unexported types need no documentation

type internal struct{}

// Good: a doc comment on the spec inside an undocumented grouped declaration
// Bad: an undocumented spec inside that group

type (
	// Grouped is documented on its spec.
	Grouped int

	Ungrouped int // want `exported type Ungrouped should have a documentation comment`
)

// Good: documented exported variable

// Version is the release version.
var Version = "1.0.0"

// Bad: exported variable without documentation

var Exposed = 42 // want `exported variable Exposed should have a documentation comment`

// Bad: constants are reported with the same message

const Limit = 10 // want `exported variable Limit should have a documentation comment`

// Good: Err-prefixed sentinel errors are exempt

var ErrNotFound = errors.New("not found")

// Good: unexported variables need no documentation

var counter = 0

// Good: the declaration's doc covers every value in the group

// Defaults documents every value in the group.
var (
	DefaultHost = "localhost"
	DefaultPort = 8080
)

// Good: a doc comment on the spec; Bad: every undocumented name in a spec

var (
	// Timeout is documented on its spec.
	Timeout = 30

	Retries, Backoff = 3, 2 // want `exported variable Retries should have a documentation comment` `exported variable Backoff should have a documentation comment`
)

// Good: nolint suppresses the diagnostic on the same line

func Suppressed() {} //nolint:exporteddoc

// Good: nolint on the preceding line suppresses the diagnostic (the directive
// is the doc comment, so the "should start with" diagnostic is suppressed)

//nolint:golint-sl
type SuppressedType struct{}

// Bad: nolint for another analyzer does not suppress

func OtherLinter() {} //nolint:nilcheck // want `exported function OtherLinter should have a documentation comment`

var _ = counter
var _ internal
