package exercises

import (
	"context"
	"sync"
)

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

// ORIGINAL (before fix): scaffold was empty goroutine bodies. Six real
// bugs on the way to this solution (full detail in MISTAKES.md): workers
// processed exactly one job instead of looping; `fn(<-jbs)` as a select-
// case argument evaluated the receive OUTSIDE the select's protection;
// the feeder never closed jbs, hanging even with no cancellation; a
// pulse-check (`if ctx.Err() != nil` before a send) was the wrong tool
// for a line that blocks on a channel operation; and close(jbs) sat
// after the loop, unreachable from the select's early return. Final fix
// for the feeder: `defer close(jbs)` at the top (runs on every exit
// path) plus `select { case jbs <- job: case <-ctx.Done(): ... }` for
// the send itself — same shape as the worker's send to results.

// ProcessAllCtx applies fn to every job using `workers` goroutines,
// stopping early (with partial results and ctx.Err()) if ctx is canceled.
func ProcessAllCtx(ctx context.Context, jobs []int, workers int, fn func(int) int) ([]int, error) {
	jbs := make(chan int)
	results := make(chan int)
	output := make([]int, 0, len(jobs))
	var wg sync.WaitGroup

	for w := 1; w <= workers; w += 1 {
		wg.Add(1)
		go func() error {
			defer wg.Done()
			for jb := range jbs { // ends when jobs is closed
				select {
				case results <- fn(jb):
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		}()
	}

	// wait and close
	go func() {
		wg.Wait()
		close(results)
	}()

	// feed the jobs channel.
	go func() error {
		defer close(jbs)
		for _, job := range jobs {
			select {
			case jbs <- job:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}()

	for result := range results {
		output = append(output, result)
	}

	return output, ctx.Err()
}
