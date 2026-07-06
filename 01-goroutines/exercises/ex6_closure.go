package exercises

// Exercise 6 — FIX THE BUG: closure capturing shared variables.
//
// A teammate read that Go 1.22 fixed the loop-variable capture bug and
// concluded closures are now always safe. Then they wrote this. The 1.22
// change made LOOP variables per-iteration — but `idx` and `job` below are
// declared OUTSIDE the loop, so there is exactly ONE of each, shared by
// every goroutine. The closures hold a live wire to those two variables
// while the loop keeps overwriting them.
//
// Typical symptom: several results are the same job processed twice, others
// are missing entirely — whatever values `idx`/`job` happened to hold when
// each goroutine finally ran. Plus data races on both variables.
//
// Your tasks:
//  1. Run the test natively and look at the wrong output. Then run it under
//     -race and read the report: which two variables race, and between whom?
//  2. Explain why the Go 1.22 fix does NOT save this code.
//  3. Fix it. (More than one clean way — pick one and defend it.)
//
// Do not change the function signature.

import "sync"

// ProcessAll runs process on every job concurrently; result[i] corresponds
// to jobs[i]. BUG: all goroutines share one idx and one job variable.
func ProcessAll(jobs []string, process func(job string) string) []string {
	results := make([]string, len(jobs))

	var wg sync.WaitGroup
	
	for i, j := range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = process(j)
		}()
	}
	wg.Wait()

	return results
}
