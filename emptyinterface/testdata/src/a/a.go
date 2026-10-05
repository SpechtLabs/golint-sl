package a

import "time"

// Bad: exported function returns interface{}
func Fetch() interface{} { // want `function "Fetch" returns interface\{\}/any; return concrete types instead`
	return nil
}

// Bad: function returns any
func lookup() any { // want `function "lookup" returns interface\{\}/any; return concrete types instead`
	return nil
}

// Bad: only the interface{} result of a multi-result function is reported
func compute() (int, any, error) { // want `function "compute" returns interface\{\}/any`
	return 0, nil, nil
}

// Good: names with an allowed prefix may return interface{}
func MarshalThing() interface{} { return nil }
func Unmarshal() any            { return nil }
func DecodeValue() any          { return nil }
func EncodeValue() any          { return nil }
func GetItem() any              { return nil }
func LoadItem() any             { return nil }
func ReadItem() any             { return nil }
func ParseInput() any           { return nil }
func ConvertInput() any         { return nil }
func WrapInput() any            { return nil }
func ValueOf() any              { return nil }

// Good: unexported functions with an allowed prefix (case-insensitive)
func getCached() any { return nil }
func parseRaw() any  { return nil }

// Good: returning a non-empty interface is fine
func Stringer() interface{ String() string } { return nil }

// Good: returning a qualified type is fine
func Now() time.Time { return time.Time{} }

// Good: no results at all
func NoResults() {}

// Bad: map[string]interface{} parameter
func Configure(opts map[string]interface{}) { // want `parameter "opts" is map\[string\]interface\{\}; consider using a struct or typed map`
	_ = opts
}

// Bad: each name in a grouped map[string]any parameter is reported
func Merge(left, right map[string]any) { // want `parameter "left" is map\[string\]any` `parameter "right" is map\[string\]any`
	_, _ = left, right
}

// Good: typed map, slice of any and plain any parameters are not flagged
func Typed(m map[string]string, s []any, v any) {
	_, _, _ = m, s, v
}

// Good: a map whose value is a qualified type is not flagged
func Durations(m map[string]time.Duration) { _ = m }

// Good: methods follow the same rules; allowed name
type Cache struct{}

func (c *Cache) Get(key string) any { return nil }

// Bad: method with a disallowed name
func (c *Cache) Item(key string) any { return nil } // want `function "Item" returns interface\{\}/any`

// Bad: struct fields holding map[string]interface{} and []interface{}
type Payload struct {
	Meta   map[string]interface{} // want `field "Meta" is map\[string\]interface\{\}; consider using a typed struct or wrapping with type-safe methods`
	Items  []interface{}          // want `field "Items" is \[\]interface\{\}; consider using a concrete element type or generics`
	A, B   []any                  // want `field "A, B" is \[\]any; consider using a concrete element type or generics`
	Labels map[string]string      // Good: typed map
	Names  []string               // Good: typed slice
	Groups map[string][]string    // Good: map of slices
	Fixed  [4]any                 // want `field "Fixed" is \[4\]any`
	Iface  interface{ Close() }   // Good: non-empty interface
}

// Good: non-struct type specs are ignored
type Bag map[string]any
type List []any

// Bad: the message names the map's actual key type
func Index(byID map[int]any) { // want `parameter "byID" is map\[int\]any; consider using a struct or typed map`
	_ = byID
}

// Bad: an alias of any is the empty interface too
type Anything = any

func Pick() Anything { // want `function "Pick" returns interface\{\}/any`
	return nil
}

// Good: a defined type is a deliberate name for the empty interface
type Value interface{}

func Current() Value { return nil }

// Good: a local type named any isn't the empty interface
func Shadowed() {
	type any struct{}
	type holder struct{ Items []any }
	_ = holder{}
}

// Good: type assertions are left to errcheck's check-type-assertions
func Assert(v any) int {
	n := v.(int)
	return n
}

// Good: nolint suppresses the diagnostic on the same line
func Suppressed() any { //nolint:emptyinterface
	return nil
}

// Good: nolint on the preceding line suppresses the diagnostic
//
//nolint:golint-sl
func SuppressedAll() any {
	return nil
}

// Bad: a nolint for a different analyzer does not suppress
func OtherLinter() any { //nolint:nilcheck // want `function "OtherLinter" returns interface\{\}/any`
	return nil
}
