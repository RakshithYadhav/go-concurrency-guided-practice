package exercises

import (
	"sync"
	"sync/atomic"
  )

// Exercise 1 — IMPLEMENT: one counter, two synchronization strategies.
//
// Both constructors must return a Counter that is exact under heavy
// concurrent use — the test hammers each with 50 goroutines x 2,000
// increments and expects precisely 100,000.
//
// Requirements:
//  1. NewMutexCounter: guard a plain int64 with a sync.Mutex. Follow the
//     convention from the notes: mutex field directly above the data it
//     guards.
//  2. NewAtomicCounter: use atomic.Int64 — no mutex anywhere.
//  3. Value() must be safe to call WHILE increments are in flight (the
//     race detector will check you on this — in the mutex version, reading
//     without the lock is still a race even though it "looks harmless").
//  4. Clean under -race, of course.
//
// After it passes, run the benchmarks and read the numbers:
//
//	go test -bench=. -run=^$ ./03-sync/exercises
//
// Which wins under contention? By how much? That ratio is the honest answer
// to "should I micro-optimize this mutex into an atomic?" for a counter —
// and the notes' answer for anything MORE complex than a counter is "you
// usually can't."

// Counter is a concurrency-safe increment-only counter.
type Counter interface {
	Inc()
	Value() int64
}

// ORIGINAL (before fix): the scaffold had no counter types at all — both
// constructors were just
//
//	func NewMutexCounter() Counter  { panic("implement me") }
//	func NewAtomicCounter() Counter { panic("implement me") }
//
// Real bugs made on the way to this solution (full detail in MISTAKES.md):
// `var` keywords inside struct fields, receiver written `(*CounterStruct cs)`,
// missing `sync/atomic` import, an unguarded read in the mutex Value(), and
// an atomic Value() that returned cs.Value() — calling itself forever.

type CounterWithMutex struct {
	mu sync.RWMutex
	count int64
}

func (cs *CounterWithMutex) Inc() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.count++
	
}

func (cs *CounterWithMutex) Value() int64{
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.count
}

type CounterWithAtomic struct {
	count atomic.Int64
}

func (cs *CounterWithAtomic) Inc() {
	cs.count.Add(1)
}

func (cs *CounterWithAtomic) Value() int64{
	return cs.count.Load()
}

// NewMutexCounter returns a Counter guarded by a sync.Mutex.
func NewMutexCounter() Counter {
	return &CounterWithMutex{}
}

// NewAtomicCounter returns a Counter backed by sync/atomic.
func NewAtomicCounter() Counter {
	return &CounterWithAtomic{}
}
