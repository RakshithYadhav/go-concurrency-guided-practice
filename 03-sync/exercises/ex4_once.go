package exercises

import "sync"

// Exercise 4 — IMPLEMENT: build sync.Once yourself.
//
// Same contract as the real thing, both guarantees (notes Section 4):
//
//  1. EXACTLY ONCE: across any number of goroutines calling Do any number
//     of times, f runs exactly once — the first call wins.
//  2. EVERYONE WAITS: no call to Do returns before f has COMPLETED. A
//     goroutine that arrives while the winner is still running f must
//     block until f finishes — otherwise it could proceed and read state
//     f hasn't initialized yet. (The test catches this specifically.)
//
// Constraints:
//  - sync.Once, sync.OnceFunc, sync.OnceValue are FORBIDDEN (that's the
//    exercise). Build from sync.Mutex and/or sync/atomic.
//  - The naive `if !o.done { o.done = true; f() }` is broken twice over —
//    it's a data race on done AND it violates guarantee 2. Start by making
//    sure you can say why, then design around it.
//  - Different f values on later calls still do NOT run (same as real
//    sync.Once: it's "once per Once value", not "once per function").
//  - Clean under -race.
//
// Hint-shaped question: what must a latecomer block ON, and what marks
// "f has completed" as opposed to "f has started"?

// MyOnce is a from-scratch sync.Once. Zero value must be ready to use.
type MyOnce struct {
	// your fields here
	mu sync.Mutex
	done bool
}

// Do runs f exactly once across all calls to this MyOnce, and does not
// return until that one run has completed.
func (o *MyOnce) Do(f func()) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.done {
		o.done = true
		f()
	}
}
