package exercises

import "sync"

// Exercise 2 — IMPLEMENT: a worker pool (the shippio shape, from scratch).
//
// ProcessAll runs fn over every job using EXACTLY `workers` goroutines —
// long-lived workers all ranging over one shared jobs channel. Not one
// goroutine per job: the test measures peak concurrency and fails if it
// ever exceeds `workers`.
//
// The moving parts you need (NOTES Section 3):
//   - a jobs channel — feed every job in, then close it (who closes it?)
//   - `workers` goroutines, each `for j := range jobs { ... }`
//   - a results channel — multiple senders, so closing it needs the
//     WaitGroup + closer-goroutine pattern from Module 2
//   - collect everything from results and return it
//
// Order of results does NOT matter (the test sorts). What matters:
//   - every job processed exactly once
//   - never more than `workers` calls to fn running at the same moment
//   - with slow fn and workers > 1, calls genuinely overlap
//   - clean under -race
//
// Assume workers >= 1. Do not change the signature.

// ORIGINAL (before fix): two bugs. `output := make([]int, len(jobs))`
// created a slice already full of `len(jobs)` zeros, and `append` added
// results AFTER those zeros — 10 jobs produced 20 results. Separately,
// `for j := range jobs { jbs <- j }` ranged over the []int like a
// channel, sending indices instead of values (same shape as ex1's bug).
// Fixed with `make([]int, 0, len(jobs))` and `for _, j := range jobs`.

// ProcessAll applies fn to every job using a pool of `workers` goroutines
// and returns all results in any order.
func ProcessAll(jobs []int, workers int, fn func(int) int) []int {
	jbs := make(chan int)
	results := make(chan int)
	output := make([]int,0, len(jobs))
	var wg sync.WaitGroup

	for w := 1; w <= workers; w+=1 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jbs {
				results <- fn(j)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for _, j := range jobs {
			jbs <- j
		}
		close(jbs)
	}()

	for res := range results {
		output = append(output, res)
	}

	return output
}
