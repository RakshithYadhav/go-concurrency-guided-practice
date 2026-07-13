package exercises

import "sync"

// Exercise 2 — FIX THE BUG: the tidy keyed mutex that isn't a mutex.
//
// This KeyedMutex looks CLEANER than the one in the notes: it deletes
// each key's mutex on unlock, so the map never grows. It passes a quick
// glance, it passes single-goroutine tests, and it is quietly, badly
// broken: under contention on ONE key, two goroutines can end up inside
// the same key's critical section AT THE SAME TIME. Mutual exclusion —
// the one job a mutex has — silently gone.
//
// Your tasks:
//  1. Find the interleaving. Three goroutines, one key. Trace who holds
//     which *sync.Mutex when B is blocked on the mutex A holds, A
//     unlocks (and DELETES), and C then calls Lock for the same key.
//     What does C find in the map? What does C create? Who's inside the
//     critical section once B's blocked Lock() finally returns?
//  2. Fix it so the SameKeySerializes test passes while the
//     DifferentKeysOverlap test keeps passing. The simple fix from
//     NOTES Section 3 is accepted. Stretch goal: reference-count
//     entries so cleanup stays AND stays correct.
//
// Do not change the method signatures.

// KeyedMutex serializes work per key. BUG: unlock's cleanup breaks
// mutual exclusion under same-key contention.
type KeyedMutex struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// NewKeyedMutex returns a ready-to-use KeyedMutex.
func NewKeyedMutex() *KeyedMutex {
	return &KeyedMutex{locks: make(map[string]*sync.Mutex)}
}

// Lock acquires the lock for key, creating it on first use.
func (k *KeyedMutex) Lock(key string) {
	k.mu.Lock()
	m, ok := k.locks[key]
	if !ok {
		m = &sync.Mutex{}
		k.locks[key] = m
	}
	k.mu.Unlock()
	m.Lock()
}

// Unlock releases the lock for key and tidies up the map.
func (k *KeyedMutex) Unlock(key string) {
	k.mu.Lock()
	m := k.locks[key]
	k.mu.Unlock()
	m.Unlock()
}

// ORIGINAL (before fix):
//
// func (k *KeyedMutex) Unlock(key string) {
// 	k.mu.Lock()
// 	m := k.locks[key]
// 	delete(k.locks, key) // BUG: tidy, and fatal
// 	k.mu.Unlock()
// 	m.Unlock()
// }
