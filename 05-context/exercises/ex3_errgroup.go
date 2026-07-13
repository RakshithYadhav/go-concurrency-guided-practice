package exercises

import (
	"context"

	"golang.org/x/sync/errgroup"
)

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

// ORIGINAL (before fix): two bugs — output = append(output, res) raced
// and scrambled order (same shape as Module 4's ProcessAll), and the
// caller's ctx was discarded in favor of context.Background(), so a
// canceled caller ctx would never reach these fetches:
//
//	func FetchAllOrFail(ctx context.Context, urls []string, fetch func(ctx context.Context, url string) (string, error)) ([]string, error) {
//		group, ectx := errgroup.WithContext(context.Background())
//		output := make([]string, 0, len(urls))
//		for _, url := range urls {
//			group.Go(func() error {
//				res, e := fetch(ectx, url)
//				if e != nil {
//					return e
//				}
//				output = append(output, res)
//				return nil
//			})
//		}
//		if err := group.Wait(); err != nil {
//			return output, err
//		}
//		return output, nil
//	}
//
// Fixed with indexed writes (output[i] = res) and errgroup.WithContext(ctx)
// — the received parameter passed straight through, not a fresh root.

// FetchAllOrFail fetches all urls concurrently; on any failure it
// returns the first error and cancels the remaining fetches.
func FetchAllOrFail(ctx context.Context, urls []string, fetch func(ctx context.Context, url string) (string, error)) ([]string, error) {
	group, ectx := errgroup.WithContext(ctx)
	output := make([]string, len(urls))
	for i, url := range urls {

		group.Go(func() error {
			res, e := fetch(ectx, url)
			if e != nil {
				return e
			}
			output[i] = res
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return output, err
	}
	return output, nil
}
