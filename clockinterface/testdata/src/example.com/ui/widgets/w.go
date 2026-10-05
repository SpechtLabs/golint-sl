// Package widgets lives under /ui/ and is exempt.
package widgets

import "time"

func Blink() {
	time.Sleep(time.Millisecond)
}
