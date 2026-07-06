package exercises

import (
	"fmt"
	"sync"
)

// Exercise 1 — IMPLEMENT: concurrent health checker.
//
// You run infrastructure with N services. Checking them one-by-one takes
// sum(latencies); checking concurrently takes ~max(latency). Build the
// concurrent version.
//
// Requirements:
//  1. Call check(target) for EVERY target, each in its own goroutine.
//  2. Return one Result per target, in the SAME ORDER as the input slice
//     (results[i] corresponds to targets[i]).
//  3. Don't return until every check has finished.
//  4. Race-free under `go test -race`. Hint: giving each goroutine its own
//     index into a pre-allocated slice is a legitimate, race-free pattern —
//     think about why.
//  5. An empty/nil targets slice returns an empty (len 0) slice, not nil-panic.
//
// Forbidden: time.Sleep, channels-as-a-crutch you can't explain. (A channel
// solution is fine if you can defend it, but WaitGroup is the natural tool.)

// Result is the outcome of checking one target.
type Result struct {
	Target string
	Err    error // nil means healthy
}

// CheckAll checks every target concurrently and returns results in input order.
func CheckAll(targets []string, check func(target string) error) []Result {
	var wg sync.WaitGroup
	results := make([]Result, len(targets))
	wg.Add(len(targets))
	
	for i, target := range targets {
		go func() {
			defer wg.Done()
			isBoom := check(target)

			res := Result{
				Target: target,
				Err:    isBoom,
			}

			results[i] = res
		}()
	}

	wg.Wait()

	fmt.Print(results)
	return results
}
