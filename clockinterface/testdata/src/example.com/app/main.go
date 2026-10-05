// Package main is exempt by its name, wherever its path points.
package main

import "time"

func run() time.Time {
	return time.Now()
}

func main() { _ = run() }
