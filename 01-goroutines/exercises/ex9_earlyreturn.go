package exercises

// Exercise 9 — FIX THE BUG: an early return that skips wg.Done().
//
// You already know `defer wg.Done()` protects against a PANIC skipping the
// decrement (see the panic/defer discussion in this module). This is the
// same failure mode without any panic at all: a perfectly ordinary early
// `return` on an invalid-input path, written BEFORE the defer is registered.
// No error, no crash — just a goroutine that quietly never calls Done().
//
// With a mix of valid and invalid ids, this hangs forever: `wg.Wait()` waits
// for every Add'd task to call Done(), but the invalid-id goroutines bail out
// long before reaching their Done() call.
//
// Your tasks:
//  1. Run the test. It has its own timeout so a hang reports as a clear
//     failure instead of freezing your terminal — but the SYMPTOM is a hang,
//     same as any real deadlock. Notice which ids make it hang.
//  2. Fix it so EVERY goroutine calls Done() no matter which path it takes,
//     while still producing no report for invalid ids.
//
// Do not change the function signature.

import "sync"

// BuildReports builds a report for every id concurrently; reports[i]
// corresponds to ids[i] (empty string for an invalid id). BUG: the early
// return on invalid input happens before wg.Done() is ever registered.
func BuildReports(ids []int, build func(id int) (string, error)) []string {
	reports := make([]string, len(ids))

	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func() {
			if id < 0 {
				return // BUG: bails out before wg.Done() is registered
			}
			defer wg.Done()
			report, _ := build(id)
			reports[i] = report
		}()
	}
	wg.Wait()

	return reports
}
