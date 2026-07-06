package exercises

import (
	"sync"
)

// Exercise 5 — IMPLEMENT: concurrent aggregation without a race.
//
// You're building a backup tool. Before uploading, it must total the size of
// every file in the batch. sizeOf is slow (imagine a network filesystem), so
// the sizes must be fetched concurrently — but the final answer is ONE number.
//
// This is one step past ex1: there, each goroutine had its own slot and you
// never combined anything. Here the results must be COMBINED, and the obvious
// move — every goroutine doing `total += sizeOf(p)` on a shared variable — is
// exactly the demo-03 lost-update race. The race detector will catch you.
//
// Requirements:
//  1. Call sizeOf(path) for EVERY path, each in its own goroutine.
//  2. Return the exact sum of all sizes.
//  3. Don't return until every sizeOf has finished.
//  4. Race-free under -race — WITHOUT using mutexes, atomics, or channels
//     (they're Modules 2-3). You already know a race-free way for goroutines
//     to hand values back: each writes to its own slot, and the combining
//     happens where only ONE goroutine is running. Where is that, and when
//     is it safe to start?
//  5. Empty/nil paths returns 0.
//
// Forbidden: time.Sleep, sync/atomic, sync.Mutex, channels.

// TotalSize returns the summed size of all paths, fetching sizes concurrently.
func TotalSize(paths []string, sizeOf func(path string) int64) int64 {
	var wg sync.WaitGroup
	sizes := make([]int64, len(paths))
	var total int64

	for i, path := range paths {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sizes[i] = sizeOf(path)
		}()
	}
	wg.Wait()

	for _, size := range sizes {
		total += size
	}

	return total
}
