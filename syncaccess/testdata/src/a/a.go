// Package a holds the syncaccess goroutine-capture cases; mutex.go holds the
// mutex-protected struct cases.
package a

import (
	"sync"
	"sync/atomic"
)

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

// Good: the goroutine takes a lock around the map write.
func lockedCapture() {
	var mu sync.Mutex
	m := make(map[string]int)
	go func() {
		mu.Lock()
		m["a"] = 1
		mu.Unlock()
	}()
	mu.Lock()
	m["b"] = 2
	mu.Unlock()
}

// Good: a read lock through a pointer, and a lock through a sync.Locker.
func readLockedCapture(mu *sync.RWMutex, l sync.Locker) {
	s := make([]int, 1)
	go func() {
		mu.RLock()
		defer mu.RUnlock()
		use(s[0])
	}()
	p := &point{}
	go func() {
		l.Lock()
		defer l.Unlock()
		p.x = 1
	}()
}

// Good: the goroutine declares its own variable with the outer one's name.
func shadowedCapture() {
	m := make(map[string]int)
	go func() {
		m := map[string]int{}
		m["a"] = 1
	}()
	m["b"] = 2
}

type holder struct{ data []int }

// Good: a field selector that shares a local variable's name is not the
// local variable.
func fieldNamedLikeLocal(t *holder) {
	data := make([]int, 1)
	go func() {
		t.data = nil
	}()
	data[0] = 1
}

// Good: from Go 1.22 on, a variable the for clause declares is a new
// variable in every iteration (oldloop.go holds the Go 1.21 cases).
func perIterationLoopVars(xs []int) {
	for i := 0; i < 3; i++ {
		go func() {
			use(i)
		}()
	}
	for k, v := range xs {
		go func() {
			use(k, v)
		}()
	}
}

// Bad: a counter the goroutine increments while the function still reads it.
func unprotectedCounter() {
	var count int
	go func() {
		count++ // want `shared variable "count" captured by goroutine without synchronization`
	}()
	use(count)
}

// Bad: a map parameter written by the goroutine and by the function.
func mapParam(m map[string]int) {
	go func() {
		m["a"] = 1 // want `shared variable "m" captured by goroutine without synchronization`
	}()
	m["b"] = 2
}

// Bad: deleting from a map parameter is a write too.
func mapParamDelete(m map[string]int) {
	go func() {
		delete(m, "a") // want `shared variable "m" captured by goroutine`
	}()
	use(len(m))
}

// Bad: a later assignment doesn't change the kind the declaration gave.
func reassignedMap() {
	m := make(map[string]int)
	go func() {
		m["a"] = 1 // want `shared variable "m" captured by goroutine`
	}()
	m = nil
	use(m)
}

// Bad: the goroutines of every iteration write the same variable.
func loopAccumulator(xs []int) {
	total := 0
	for _, x := range xs {
		go func() {
			total += x // want `shared variable "total" captured by goroutine`
		}()
	}
}

// Bad: the function uses the variable in the next iteration of the loop.
func loopReadBeforeGo(xs []int) {
	var last int
	for _, x := range xs {
		use(last)
		go func(x int) {
			last = x // want `shared variable "last" captured by goroutine`
		}(x)
	}
}

// Good: the function waits for the goroutine before reading the result.
func waitGroupResult() int {
	var wg sync.WaitGroup
	var result int
	wg.Add(1)
	go func() {
		defer wg.Done()
		result = 42
	}()
	wg.Wait()
	return result
}

// Good: closing a channel orders the write before the read.
func channelResult() int {
	done := make(chan struct{})
	var result int
	go func() {
		result = 42
		close(done)
	}()
	<-done
	return result
}

// Good: a call through a function value may synchronize.
func callbackResult(done func()) int {
	var result int
	go func() {
		result = 42
		done()
	}()
	return result
}

// Good: the function doesn't use the variable after starting the goroutine.
func writtenOnlyByGoroutine() {
	count := 0
	count++
	go func() {
		count++
		use(count)
	}()
}

// Good: a slice parameter only read, by the goroutine and by the function.
func readOnlyParam(xs []int) {
	go func() {
		use(xs[0])
	}()
	use(xs[1])
}

// Bad: a range clause inside the goroutine assigns the captured variable.
func rangeAssignInGoroutine(xs []int) {
	var last int
	go func() {
		for _, last = range xs { // want `shared variable "last" captured by goroutine`
		}
	}()
	use(last)
}

// Bad: a conversion is not synchronization, and a loop without an init
// statement declares no loop variable.
func conversionInLoop(n int) {
	var f float64
	for n > 0 {
		go func() {
			f = float64(n) // want `shared variable "f" captured by goroutine`
		}()
		n--
	}
	for k := range n {
		go func() {
			f = float64(k) // want `shared variable "f" captured by goroutine`
		}()
	}
	use(f)
}

var packageCounter int

// Good: package-level variables are not captured locals.
func packageLevel() {
	go func() {
		packageCounter++
	}()
	use(packageCounter)
}

// Good: receiving from a channel synchronizes.
func receiveResult(start <-chan struct{}) int {
	var result int
	go func() {
		<-start
		result = 1
	}()
	return result
}

// Good: ranging over a channel synchronizes.
func rangeChannelResult(in <-chan int) int {
	var result int
	go func() {
		for v := range in {
			result = v
		}
	}()
	return result
}

// Good: an atomic operation synchronizes.
func atomicResult(flag *atomic.Bool) int {
	var result int
	go func() {
		result = 1
		flag.Store(true)
	}()
	return result
}

// Good: goroutines that are waited for write distinct slice elements.
func fanOutSlice(xs []int) []int {
	var wg sync.WaitGroup
	out := make([]int, len(xs))
	for i, x := range xs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = x * 2
		}()
	}
	wg.Wait()
	return out
}

// Good: a declared slice only read by a goroutine the function waits for.
func waitedReader(xs []int) {
	var roots []int
	roots = append(roots, xs...)
	done := make(chan struct{})
	go func() {
		use(roots)
		close(done)
	}()
	<-done
}

// Bad: the goroutines of a loop write the same map, even though the function
// waits for them.
func fanOutMap(xs []string) map[string]int {
	var wg sync.WaitGroup
	counts := map[string]int{}
	for _, x := range xs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counts[x]++ // want `shared variable "counts" captured by goroutine`
		}()
	}
	wg.Wait()
	return counts
}

// Good: a value a method call returns is not tracked as a reference, nor is
// a var declaration initialized without an explicit type.
func untypedInitializers() {
	got := (&plain{n: 1}).Get()
	var out = make([]int, 1)
	go func() {
		use(got, out)
	}()
}

// Bad: a function literal the goroutine calls is inspected, not assumed to
// synchronize.
func deferredRecover() {
	var count int
	go func() {
		defer func() {
			_ = recover()
		}()
		count++ // want `shared variable "count" captured by goroutine`
	}()
	use(count)
}
