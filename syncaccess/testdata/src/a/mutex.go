package a

import (
	"fmt"
	"sync"
)

type ID string

type plain struct{ n int }

func (p *plain) Get() int { return p.n }

type Counter struct {
	mu    sync.Mutex
	count int
	inner struct{ hits int }
}

// Good: locks before touching fields.
func (c *Counter) Increment() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}

// Bad: reads a field without locking.
func (c *Counter) Value() int { // want `method "Value" on type with mutex accesses fields without Lock\(\); consider adding synchronization`
	return c.count
}

// Bad: nested field access through the receiver.
func (c *Counter) Hits() int { // want `method "Hits" on type with mutex accesses fields without Lock`
	return c.inner.hits
}

// Good: only the mutex itself is touched.
func (c *Counter) Locker() sync.Locker {
	return &c.mu
}

// Good: an unnamed receiver cannot access fields.
func (*Counter) Name() string { return "counter" }

// Good: selectors on other identifiers are not receiver field accesses.
func (c *Counter) Describe() string {
	return fmt.Sprint("counter")
}

// Good: suppressed with a nolint directive.
//
//nolint:syncaccess
func (c *Counter) Unsafe() int {
	return c.count
}

type Cache struct {
	mutex      *sync.RWMutex
	stateMutex sync.Mutex
	data       map[string]string
}

// Good: RLock counts as locking.
func (c Cache) Get(k string) string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.data[k]
}

// Bad: value receiver reading a field without locking.
func (c Cache) Len() int { // want `method "Len" on type with mutex accesses fields without Lock`
	return len(c.data)
}

// Good: fields whose names contain "mutex" are skipped.
func (c Cache) Locks() (*sync.RWMutex, *sync.Mutex) {
	return c.mutex, &c.stateMutex
}

type Embedded struct {
	sync.Mutex
	n int
}

// Good: an embedded mutex locked through the promoted method.
func (e *Embedded) Set(n int) {
	e.Lock()
	defer e.Unlock()
	e.n = n
}

// Bad: an embedded mutex that is never locked.
func (e *Embedded) N() int { // want `method "N" on type with mutex accesses fields without Lock`
	return e.n
}

type Box[T any] struct {
	mu sync.Mutex
	v  T
}

// Good: generic receivers are not matched by name.
func (b *Box[T]) Value() T { return b.v }
