// Demo 02: fan-out / fan-in — one slow stage vs 8 copies of it.
//
//	go run ./04-patterns/demo/02-fanout
//
// The "slow stage" takes 10ms per item (a pretend API call). 24 items:
//   - serial: one worker, ~240ms
//   - fan-out 8 wide: eight workers sharing the SAME input channel, ~30ms
// Also watch the output order in the fan-out run: scrambled. Results come
// out in completion order, not input order. That's the price.
package main

import (
	"fmt"
	"sync"
	"time"
)

func generate(n int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for i := 1; i <= n; i++ {
			out <- i
		}
	}()
	return out
}

// slowDouble is the bottleneck stage: 10ms per item.
func slowDouble(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			time.Sleep(10 * time.Millisecond)
			out <- v * 2
		}
	}()
	return out
}

// merge is fan-in: many channels back into one (Module 2 ex4, reused).
func merge(chans ...<-chan int) <-chan int {
	out := make(chan int)
	var wg sync.WaitGroup
	wg.Add(len(chans))
	for _, ch := range chans {
		go func(c <-chan int) {
			defer wg.Done()
			for v := range c {
				out <- v
			}
		}(ch)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func main() {
	const items = 24

	// --- serial: one copy of the slow stage ---
	start := time.Now()
	for range slowDouble(generate(items)) {
	}
	fmt.Printf("serial (1 worker):   %d items in %v\n", items, time.Since(start).Round(time.Millisecond))

	// --- fan-out: 8 copies reading the SAME input channel ---
	start = time.Now()
	in := generate(items)
	workers := make([]<-chan int, 8)
	for i := range workers {
		workers[i] = slowDouble(in) // all 8 share `in`
	}
	var results []int
	for v := range merge(workers...) {
		results = append(results, v)
	}
	fmt.Printf("fan-out (8 workers): %d items in %v\n", items, time.Since(start).Round(time.Millisecond))
	fmt.Printf("fan-out output order (input was 1,2,3,...): %v\n", results)
}
