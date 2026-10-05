// Package cmd is CLI code: fmt.Print* is user output there, not logging.
package cmd

import (
	"fmt"
	"log"
)

// Good: fmt.Println is allowed in CLI packages; stdlib log is still banned.
func printUsage() {
	fmt.Println("usage: tool [flags]")
	log.Printf("still banned") // want `stdlib log is banned`
}
