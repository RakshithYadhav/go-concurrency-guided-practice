// Demo 02: RWMutex lets readers overlap; Mutex serializes them.
//
//	go run ./03-sync/demo/02-rwmutex
//
// 50 goroutines each "read" a config (the read itself takes ~2ms). Under a
// plain Mutex the reads run one at a time: ~50 x 2ms. Under RWMutex's RLock
// they all overlap: close to ~2ms total. That gap is the entire sales pitch
// of RWMutex — and it only exists because these are READS.
//
// Caveat from the notes: on write-heavy or uncontended workloads, RWMutex's
// extra bookkeeping makes it SLOWER than Mutex. Measure before upgrading.
package main

import (
	"fmt"
	"sync"
	"time"
)

const readers = 50

func simulateRead() { time.Sleep(2 * time.Millisecond) }

func main() {
	var wg sync.WaitGroup

	// --- plain Mutex: readers serialize ---
	var mu sync.Mutex
	start := time.Now()
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			simulateRead()
			mu.Unlock()
		}()
	}
	wg.Wait()
	fmt.Printf("Mutex:   %d readers took %v (one at a time)\n", readers, time.Since(start))

	// --- RWMutex: readers overlap ---
	var rw sync.RWMutex
	start = time.Now()
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rw.RLock()
			simulateRead()
			rw.RUnlock()
		}()
	}
	wg.Wait()
	fmt.Printf("RWMutex: %d readers took %v (all at once)\n", readers, time.Since(start))
}
