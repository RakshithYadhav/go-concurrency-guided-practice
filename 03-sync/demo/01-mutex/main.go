// Demo 01: the Module-1 racy counter, fixed with a mutex.
//
//	go run ./03-sync/demo/01-mutex
//
// Same setup as 01-goroutines/demo/03-race: 10 goroutines x 10,000
// increments. The racy version loses updates silently; the mutex version is
// exact, every run. Also run the racy half under the detector if you like:
// it's the same report you've seen before.
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
	// --- the bug, again (Module 1 demo 03) ---
	var racy int
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				racy++ // LOAD + ADD + STORE, interleavable — updates vanish
			}
		}()
	}
	wg.Wait()

	// --- the fix ---
	var (
		mu    sync.Mutex
		exact int
	)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < increments; j++ {
				mu.Lock()
				exact++ // only ONE goroutine can be between Lock and Unlock
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	want := goroutines * increments
	fmt.Printf("want %d\n", want)
	fmt.Printf("racy  counter: %6d  (lost %d)\n", racy, want-racy)
	fmt.Printf("mutex counter: %6d  (lost %d — every run, guaranteed)\n", exact, want-exact)
}
