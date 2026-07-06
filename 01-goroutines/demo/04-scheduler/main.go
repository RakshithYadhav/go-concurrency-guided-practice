// Demo 04: observing the runtime scheduler.
//
//	go run ./01-goroutines/demo/04-scheduler
//
// Then rerun pinned to one P and compare wall time of the CPU phase:
//
//	GOMAXPROCS=1 go run ./01-goroutines/demo/04-scheduler   (bash)
//	$env:GOMAXPROCS='1'; go run ./01-goroutines/demo/04-scheduler   (PowerShell)
//
// What to look for:
//  1. GOMAXPROCS = number of Ps = max goroutines running Go code in parallel.
//  2. Spawning 100k goroutines takes milliseconds and modest memory — they
//     start with ~2KB stacks. Try 100k OS threads sometime (don't).
//  3. CPU-bound work scales with Ps: with GOMAXPROCS=1 the busy-loop phase
//     runs ~Ncores times slower — concurrency without parallelism.
package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

func main() {
	fmt.Println("GOMAXPROCS:", runtime.GOMAXPROCS(0)) // 0 = just report
	fmt.Println("NumCPU:    ", runtime.NumCPU())

	// --- 1. Goroutines are cheap: spawn 100,000 of them ---
	const n = 100_000
	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	var wg sync.WaitGroup
	release := make(chan struct{}) // hold them all alive at once
	start := time.Now()
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-release // park until released (parked G ≈ free)
		}()
	}
	spawn := time.Since(start)
	runtime.ReadMemStats(&memAfter)
	fmt.Printf("spawned %d goroutines in %v (now live: %d, ~%d KB each)\n",
		n, spawn, runtime.NumGoroutine(),
		(memAfter.Sys-memBefore.Sys)/n/1024)

	close(release) // release them all; closing a channel wakes every receiver
	wg.Wait()

	// --- 2. CPU-bound work: parallelism = GOMAXPROCS ---
	workers := runtime.GOMAXPROCS(0)
	start = time.Now()
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			busy := 0
			for j := 0; j < 500_000_000; j++ { // pure CPU burn
				busy += j
			}
			_ = busy
		}()
	}
	wg.Wait()
	fmt.Printf("%d CPU-bound workers finished in %v (rerun with GOMAXPROCS=1 and compare)\n",
		workers, time.Since(start))
}
