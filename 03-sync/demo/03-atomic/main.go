// Demo 03: the same counter three ways — racy, mutex, atomic.
//
//	go run ./03-sync/demo/03-atomic
//
// Correctness: racy is wrong, the other two are exact.
// Speed: atomic beats mutex under contention — the CPU instruction does the
// whole read-modify-write indivisibly, with no parking or scheduling.
// (Rough wall-clock numbers only; ex1's benchmarks measure this properly.)
package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	goroutines = 8
	increments = 200_000
)

func run(name string, inc func()) {
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				inc()
			}
		}()
	}
	wg.Wait()
	fmt.Printf("%-7s took %v", name, time.Since(start))
}

func main() {
	want := goroutines * increments
	fmt.Printf("want: %d\n\n", want)

	var racy int
	run("racy", func() { racy++ })
	fmt.Printf("  -> got %d (lost %d)\n", racy, want-racy)

	var (
		mu      sync.Mutex
		guarded int
	)
	run("mutex", func() { mu.Lock(); guarded++; mu.Unlock() })
	fmt.Printf("  -> got %d\n", guarded)

	var counter atomic.Int64
	run("atomic", func() { counter.Add(1) })
	fmt.Printf("  -> got %d\n", counter.Load())
}
