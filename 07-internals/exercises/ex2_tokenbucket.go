package exercises

import "context"

// Exercise 2 — GAUNTLET (25 min): build a token-bucket rate limiter.
//
// You USED x/time/rate in Module 6. Now build the bucket yourself.
// No x/time imports — sync, time, context, arithmetic.
//
// The model (NOTES Module 6, Section 2): a bucket holds at most
// `burst` tokens and refills continuously at `perSecond` tokens per
// second. Every operation spends one token. A NEW bucket starts FULL.
//
//   - Allow() — non-blocking. A token is available? Spend it, return
//     true. None? Return false IMMEDIATELY. Never, ever sleeps.
//   - Wait(ctx) — blocking. Spend a token now if one's there;
//     otherwise wait until one accrues — UNLESS ctx dies first, in
//     which case return ctx's error promptly (Module 5's rule: no
//     wait may outlive its context).
//
// Implementation hint you're allowed (it's a design fact, not a
// solution): you do NOT need a background goroutine dripping tokens.
// The standard trick is LAZY refill — remember when you last updated,
// and on each call compute how many tokens have accrued since then
// (elapsed × rate, capped at burst). Do the math, then decide.
//
// Think before you type:
//   - The shape: shared mutable state (token count + last-refill
//     time) touched by many goroutines. Module 3 says what guards it.
//   - The exits: Wait blocks — on WHAT? Whatever it is, ctx.Done()
//     must be able to interrupt it.
//
// Assume perSecond >= 1, burst >= 1. Do not change the signatures.

// TokenBucket is a rate limiter: at most `burst` tokens held, refilled
// at perSecond tokens per second.
type TokenBucket struct {
	// your fields here
}

// NewTokenBucket returns a bucket that starts full.
func NewTokenBucket(perSecond int, burst int) *TokenBucket {
	panic("implement me")
}

// Allow spends a token if one is available. It never blocks.
func (tb *TokenBucket) Allow() bool {
	panic("implement me")
}

// Wait blocks until a token can be spent or ctx is done.
func (tb *TokenBucket) Wait(ctx context.Context) error {
	panic("implement me")
}
