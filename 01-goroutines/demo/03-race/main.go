// Demo 03: a deliberate data race.
//
// Run it BOTH ways and compare:
//
//	go run ./01-goroutines/demo/03-race          // wrong total, silently
//	go run -race ./01-goroutines/demo/03-race    // WARNING: DATA RACE + stacks
//
// 10 goroutines x 10,000 increments should total 100,000. It won't (usually).
// count++ is load → add → store; two goroutines interleave those steps and
// updates vanish. Note the plain run gives NO error — just a wrong number.
// That's what makes races the worst class of bug: silent corruption.
//
// Fixes (Module 3 covers them properly): sync.Mutex, or sync/atomic —
// atomic.Int64's Add method makes the read-modify-write one indivisible step.
package main

import (
	"fmt"
	"sync"
)

const (
	goroutines = 10
	increments = 10_000
)

func main() {
	var count int // shared, unsynchronized — the bug

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				count++ // RACE: unsynchronized read-modify-write
			}
		}()
	}
	wg.Wait()

	fmt.Printf("expected %d, got %d (lost %d updates)\n",
		goroutines*increments, count, goroutines*increments-count)
}
