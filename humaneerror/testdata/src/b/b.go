// Package b imports a humane-style package under a custom alias that is not
// one of the analyzer's well-known identifiers.
package b

import apperr "example.com/humanelib"

// Good: advice present.
func goodNew() apperr.Error {
	return apperr.New("disk full", "free space on /var")
}

// Bad: the alias is resolved through the import path containing "humane".
func badWrap(err error) apperr.Error {
	return apperr.Wrap(err, "write failed") // want `humane.Wrap\(\) should include at least one advice string`
}
