package exercises

import "context"

// Exercise 3 — IMPLEMENT: parallel fetch, first error cancels the rest.
//
// FetchAllOrFail fetches every URL concurrently and returns the results
// IN INPUT ORDER (slots — Module 1, again). But this is production now:
//
//   - if EVERY fetch succeeds → (results, nil)
//   - if ANY fetch fails → return that error, AND the other in-flight
//     fetches must be canceled promptly instead of running to the end.
//     The test measures wall time: four 800ms fetches alongside one that
//     fails at 30ms must ALL be done (canceled) well before 800ms.
//
// Requirements:
//   - use errgroup.WithContext (import "golang.org/x/sync/errgroup") —
//     hand-rolling this with WaitGroup+channels is possible, but the
//     lesson here is the tool everyone uses in production
//   - pass the GROUP's ctx to every fetch call — that's the only way the
//     first failure can reach the others (trace why: who cancels that
//     ctx, and when?)
//   - results slice in input order; write into results[i], no locks
//     needed — each goroutine owns its slot
//
// Do not change the signature.

// FetchAllOrFail fetches all urls concurrently; on any failure it
// returns the first error and cancels the remaining fetches.
func FetchAllOrFail(ctx context.Context, urls []string, fetch func(ctx context.Context, url string) (string, error)) ([]string, error) {
	panic("implement me")
}
