// Demo 02: keyed mutex — same key serializes, different keys overlap.
//
//	go run ./06-patterns2/demo/02-keyedmutex
//
// Four jobs, each holding its key's lock for 100ms:
//   - all four on the SAME key  → they queue: ~400ms total
//   - four DIFFERENT keys       → they overlap: ~100ms total
// One global mutex would make BOTH cases take 400ms. That gap is the
// entire reason the keyed mutex exists.
package main

import (
	"fmt"
	"sync"
	"time"
)

// KeyedMutex: one lock per key, created on demand. (Correct version —
// exercise 2 hands you a subtly broken one to fix.)
type KeyedMutex struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func NewKeyedMutex() *KeyedMutex {
	return &KeyedMutex{locks: make(map[string]*sync.Mutex)}
}

func (k *KeyedMutex) Lock(key string) {
	k.mu.Lock()
	m, ok := k.locks[key]
	if !ok {
		m = &sync.Mutex{}
		k.locks[key] = m
	}
	k.mu.Unlock() // release the MAP lock before the long wait
	m.Lock()
}

func (k *KeyedMutex) Unlock(key string) {
	k.mu.Lock()
	m := k.locks[key]
	k.mu.Unlock()
	m.Unlock()
}

func run(km *KeyedMutex, keys []string) time.Duration {
	start := time.Now()
	var wg sync.WaitGroup
	for i, key := range keys {
		wg.Add(1)
		go func(n int, k string) {
			defer wg.Done()
			km.Lock(k)
			defer km.Unlock(k)
			time.Sleep(100 * time.Millisecond) // "work" holding the key
			fmt.Printf("  job %d on key %q done at %v\n", n, k, time.Since(start).Round(10*time.Millisecond))
		}(i+1, key)
	}
	wg.Wait()
	return time.Since(start)
}

func main() {
	km := NewKeyedMutex()

	fmt.Println("four jobs, SAME key (\"msft\") — they must take turns:")
	total := run(km, []string{"msft", "msft", "msft", "msft"})
	fmt.Printf("total: %v\n\n", total.Round(10*time.Millisecond))

	fmt.Println("four jobs, DIFFERENT keys — nothing to wait for:")
	total = run(km, []string{"msft", "aapl", "goog", "amzn"})
	fmt.Printf("total: %v\n", total.Round(10*time.Millisecond))
}
