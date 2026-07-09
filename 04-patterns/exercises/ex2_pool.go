package exercises

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

// ProcessAll applies fn to every job using a pool of `workers` goroutines
// and returns all results in any order.
func ProcessAll(jobs []int, workers int, fn func(int) int) []int {
	panic("implement me")
}
