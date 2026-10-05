// Package tool lives under /cmd/ and is exempt.
package tool

import "time"

func Run() {
	_ = time.Now()
}
