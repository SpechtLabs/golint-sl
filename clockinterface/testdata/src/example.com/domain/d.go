// Package domain's path ends in "main", but the package is not package main.
package domain

import "time"

// Bad: the path suffix does not make this an entry point
func Expire() time.Time {
	return time.Now() // want `direct time.Now\(\) call in business logic`
}
