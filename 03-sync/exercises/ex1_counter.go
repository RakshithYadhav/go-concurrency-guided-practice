package exercises

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

// NewMutexCounter returns a Counter guarded by a sync.Mutex.
func NewMutexCounter() Counter {
	panic("implement me")
}

// NewAtomicCounter returns a Counter backed by sync/atomic.
func NewAtomicCounter() Counter {
	panic("implement me")
}
