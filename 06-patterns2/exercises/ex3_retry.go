package exercises

import (
	"context"
	"math/rand"
	"time"
)

// Exercise 3 — IMPLEMENT: retry with backoff, jitter, and a budget.
//
// Retry runs op up to `attempts` times, returning nil on the first
// success. Between attempts it waits — and HOW it waits is the whole
// exercise (NOTES Section 4):
//
//   - exponential backoff: the wait before retry #n is base * 2^(n-1) —
//     base, then 2*base, then 4*base, ...
//   - plus jitter: add a random extra of up to 50% of that wait
//     (math/rand is fine)
//   - ctx-aware: every wait must be a select against ctx.Done(). If ctx
//     dies mid-wait, return ctx.Err() immediately — a canceled request
//     must never sit out a 4-second backoff first. A plain time.Sleep
//     fails the cancellation test.
//   - budget: after the LAST failed attempt, return that error without
//     waiting again (no pointless final sleep).
//
// Contract:
//   - op succeeds on try k ≤ attempts → nil, exactly k calls
//   - all attempts fail → the last error, exactly `attempts` calls
//   - ctx canceled during a wait → ctx.Err(), no further calls to op
//
// Assume attempts >= 1. Do not change the signature.

// Retry runs op up to attempts times with exponential backoff + jitter
// between tries, honoring ctx during waits.
func Retry(ctx context.Context, attempts int, base time.Duration, op func() error) error {
	var err error

	for i := 0; i < attempts; i++ {
		if err = op(); err == nil {
			return nil
		}
		if i == attempts-1 {
			break
		}

		wait := base * (1 << i)
		wait += time.Duration(rand.Int63n(int64(wait) / 2))
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}

// ORIGINAL (before fix):
//
// func Retry(ctx context.Context, attempts int, base time.Duration, op func() error) error {
// 	panic("implement me")
// }
