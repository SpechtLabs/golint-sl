package a

import "log"

// Good: test files are skipped entirely.
func helperInTestFile() {
	log.Println("test output")
}
