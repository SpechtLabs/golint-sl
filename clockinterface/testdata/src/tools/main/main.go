// Package main lives under a path ending in "main" and is exempt.
package main

import "time"

func run() {
	_ = time.Now()
	time.Sleep(time.Second)
}

func main() { run() }
