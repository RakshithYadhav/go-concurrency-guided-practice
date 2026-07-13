package exercises

import (
	"context"
	"golang.org/x/time/rate"
)

// Exercise 1 — IMPLEMENT: a rate-limited fetcher.
//
// FetchAllPaced fetches every URL — but never faster than `perSecond`
// calls per second, no matter how many URLs there are. This is the
// polite client from NOTES Section 2: the downstream API has a quota,
// and it's YOUR job not to blow through it.
//
// Contract:
//   - calls to fetch are PACED: consecutive calls at least ~1/perSecond
//     apart (the test records call times and measures the gaps; the
//     first call may fire immediately)
//   - results come back in INPUT ORDER: results[i] belongs to urls[i]
//   - ctx canceled mid-run → stop promptly, return (partial results so
//     far, ctx.Err()); waiting for the next token must not outlive ctx
//   - ctx never canceled → (all results, nil)
//
// Tool choice is yours (NOTES Section 2): a time.Ticker you range over,
// or rate.Limiter.Wait(ctx) — the vendored golang.org/x/time/rate is
// available. Note: with pacing this strict, fetching sequentially in one
// goroutine is perfectly fine — think about whether you even need
// concurrency here before reaching for goroutines.
//
// Assume perSecond >= 1. Do not change the signature.

// FetchAllPaced fetches all urls at no more than perSecond calls per
// second, returning results in input order.
func FetchAllPaced(ctx context.Context, urls []string, perSecond int, fetch func(string) string) ([]string, error) {
	limiter := rate.NewLimiter(rate.Limit(perSecond), 1)
	output := make([]string, 0, len(urls))

	for _, url := range urls {
		err := limiter.Wait(ctx)
		if err != nil {
			return output, err
		}
		output = append(output, fetch(url))
	}

	return output, nil
}
