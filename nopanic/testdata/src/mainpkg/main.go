// Package main is a program, so panics and fatal logs are allowed.
package main

import "log"

func main() {
	log.Fatal("main packages may exit")
	panic("and panic")
}
