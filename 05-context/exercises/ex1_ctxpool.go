package exercises

import "context"

// Exercise 1 — IMPLEMENT: a worker pool that can be told to stop.
//
// This is Module 4's ProcessAll with one production upgrade: a context.
// While ctx is alive it behaves exactly like before — `workers`
// goroutines chew through the jobs, results come back in any order.
// When ctx is canceled mid-run, the pool LETS GO:
//
//   - the feeder stops handing out new jobs
//   - workers finish (at most) the job in their hands, then exit
//   - ProcessAllCtx returns promptly: the results gathered so far,
//     plus ctx.Err()
//   - and NOTHING leaks — no goroutine may be left blocked on a send
//     after the early return (the test counts, like Module 4 ex3)
//
// Contract:
//   - ctx never canceled → (all results in any order, nil)
//   - ctx canceled mid-run → (partial results, ctx.Err()) — and the
//     return must happen within a few job-lengths, not after the whole
//     queue is done
//
// Everything you need is Module 4's pool plus NOTES Section 4: every
// channel send gets a second exit via select + ctx.Done(). Ask of every
// goroutine you start: "when ctx dies, what unblocks THIS one?"
//
// Assume workers >= 1. Do not change the signature.

// ProcessAllCtx applies fn to every job using `workers` goroutines,
// stopping early (with partial results and ctx.Err()) if ctx is canceled.
func ProcessAllCtx(ctx context.Context, jobs []int, workers int, fn func(int) int) ([]int, error) {
	panic("implement me")
}
