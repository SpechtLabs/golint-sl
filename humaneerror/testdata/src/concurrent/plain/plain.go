// Package plain has no framework callbacks, so every fmt.Errorf is reported
// no matter what other packages are analyzed at the same time.
package plain

import "fmt"

var errA = fmt.Errorf("a: %d", 1) // want `avoid fmt.Errorf\(\); use humane.Wrap`

func load() error { return fmt.Errorf("load: %w", errA) } // want `avoid fmt.Errorf\(\); use humane.Wrap`

var errB = fmt.Errorf("b: %d", 2) // want `avoid fmt.Errorf\(\); use humane.Wrap`

func store() error { return fmt.Errorf("store: %w", errB) } // want `avoid fmt.Errorf\(\); use humane.Wrap`

var errC = fmt.Errorf("c: %d", 3) // want `avoid fmt.Errorf\(\); use humane.Wrap`

func parse() error { return fmt.Errorf("parse: %w", errC) } // want `avoid fmt.Errorf\(\); use humane.Wrap`
