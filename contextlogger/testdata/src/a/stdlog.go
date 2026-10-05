package a

import (
	"context"
	"log"
)

// Bad: the standard library's global logger
func StdLog(ctx context.Context) {
	log.Print("x") // want `function has context parameter but uses global logger`
}

// Bad: Printf matches only the log.Printf pattern, not log.Print as well, so
// it is reported once
func Printf(ctx context.Context) {
	log.Printf("x") // want `function has context parameter but uses global logger`
}

// Bad: likewise Println and Fatalf
func Println(ctx context.Context) {
	log.Println("x") // want `function has context parameter but uses global logger`
	log.Fatalf("x")  // want `function has context parameter but uses global logger`
}
