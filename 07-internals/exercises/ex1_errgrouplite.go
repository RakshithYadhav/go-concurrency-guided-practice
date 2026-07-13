package exercises

import "context"

// Exercise 1 — GAUNTLET (25 min): build errgroup from scratch.
//
// You USED x/sync/errgroup in Module 5. Now build the core of it
// yourself. No x/sync imports — sync, context, and your own hands.
//
// The contract (same as the real thing):
//
//   - Go(fn) runs fn in its own goroutine.
//   - Wait() blocks until EVERY fn started with Go has returned, then
//     returns the FIRST error that occurred (or nil if none did).
//     Errors after the first are thrown away.
//   - WithContext(ctx) returns a Group plus a derived context that is
//     CANCELED the moment any fn returns a non-nil error — so
//     in-flight siblings can see the failure and stop early.
//   - The zero value of Group must be usable too: var g Group, then
//     Go/Wait work — there's just no context to cancel.
//
// Think before you type (NOTES Section 5's narration drill):
//   - The shape: what pieces does this need? Something that waits for
//     N goroutines. Something that remembers one error. Something
//     that cancels.
//   - The ownership: who writes the error? Potentially many goroutines
//     at once — Module 3 told you what that means.
//   - The exits: what unblocks Wait? Every fn returning — including
//     the failed one's siblings.
//
// Do not change the signatures.

// Group runs related goroutines and collects their first error.
type Group struct {
	// your fields here
}

// WithContext returns a Group and a context derived from ctx that is
// canceled the first time a function passed to Go returns an error.
func WithContext(ctx context.Context) (*Group, context.Context) {
	panic("implement me")
}

// Go runs fn in a new goroutine.
func (g *Group) Go(fn func() error) {
	panic("implement me")
}

// Wait blocks until all functions started with Go have returned, then
// returns the first non-nil error among them (or nil).
func (g *Group) Wait() error {
	panic("implement me")
}
