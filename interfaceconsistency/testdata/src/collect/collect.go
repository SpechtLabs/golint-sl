// Package collect is input for AnalyzeInterfaces: two interfaces, one struct.
package collect

// Reader reads.
type Reader interface {
	Read() string
}

type writer interface {
	write(s string)
}

// File is a concrete type, not an interface.
type File struct{}
