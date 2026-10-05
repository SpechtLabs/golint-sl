// Package a holds the syncaccess goroutine-capture cases; mutex.go holds the
// mutex-protected struct cases.
package a

// Good: a goroutine outside any function declaration has no enclosing locals.
var _ = func() int {
	go func() {}()
	return 0
}()

func use(...any) {}

func worker(m map[string]int) {}

type point struct{ x, y int }

// Bad: maps, slices and pointers captured by a goroutine.
func sharedCaptures(n int) {
	m := make(map[string]int)
	s := make([]int, n)
	lit := []int{1, 2}
	ml := map[string]int{}
	p := &point{}
	go func() {
		m["a"] = 1 // want `shared variable "m" captured by goroutine without synchronization`
	}()
	go func() {
		s[0] = 1 // want `shared variable "s" captured by goroutine`
	}()
	go func() {
		lit[0] = 1 // want `shared variable "lit" captured by goroutine`
	}()
	go func() {
		ml["a"] = 1 // want `shared variable "ml" captured by goroutine`
	}()
	go func() {
		p.x = 1 // want `shared variable "p" captured by goroutine`
	}()
	m["b"] = 2
	s[0], lit[0], ml["b"], p.y = 2, 2, 2, 2
}

// Bad: reference types declared with var.
func declaredCaptures() {
	var p *point
	var m map[string]int
	var s []int
	go func() {
		use(p, m, s) // want `shared variable "p"` `shared variable "m"` `shared variable "s"`
	}()
	use(p, m, s)
}

// Bad: a loop variable declared outside the loop is shared by every
// iteration, so the goroutine races with the loop's increment.
func sharedLoopCounter() {
	var i int
	for i = 0; i < 3; i++ {
		go func() {
			use(i) // want `loop variable "i" captured by goroutine; this may cause unexpected behavior - pass as parameter instead`
		}()
	}
}

// Bad: range key and value assigned to outer variables.
func sharedRangeVars(m map[string]int) {
	var k string
	var v int
	for k, v = range m {
		go func() {
			use(k) // want `loop variable "k" captured by goroutine`
		}()
		go func() {
			use(v) // want `loop variable "v" captured by goroutine`
		}()
	}
}

// Good: the shared counter is passed as a parameter.
func loopCounterAsParam() {
	var i int
	for i = 0; i < 3; i++ {
		go func(i int) {
			use(i)
		}(i)
	}
}

// Good: channels are safe to share; scalars and values of other kinds are
// not tracked as references; a call that is not a function literal is skipped.
func notFlagged(src <-chan int, n int) {
	ch := make(chan int)
	var done chan struct{}
	neg := -n
	recv := <-src
	pt := point{}
	made := make(chan bool, 1)
	sum := add(1, 2)
	str := string(rune(n))
	m := make(map[string]int)
	var plain int
	go func() {
		ch <- neg + recv + pt.x + sum + len(str) + plain
		close(done)
		made <- true
	}()
	go worker(m)
}

// Good: loop variables whose loop finished before the goroutine started.
func afterLoops(xs []int) {
	var i, k int
	for i = 0; i < 3; i++ {
	}
	for k = range xs {
	}
	for range xs {
	}
	for i < 5 {
		i++
	}
	go func() {
		use(i, k)
	}()
}

func add(a, b int) int { return a + b }

// Good: suppressed with a nolint directive.
func suppressedCapture() {
	m := make(map[string]int)
	go func() {
		m["a"] = 1 //nolint:syncaccess
	}()
}
