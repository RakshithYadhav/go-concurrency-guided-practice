package exercises

// Exercise 4 — IMPLEMENT: fan-in (merge N channels into one).
//
// You have several producers, each streaming values on its own channel
// (imagine: one Generate() per data source). Downstream code wants ONE
// channel with everything on it. Merge is that adapter, and it's a
// foundational pattern — Module 4's pipelines are built out of it.
//
// Requirements:
//  1. Every value from every input channel appears on the returned channel
//     (any order across inputs; nothing lost, nothing duplicated).
//  2. The returned channel is CLOSED after — and only after — ALL input
//     channels are closed and drained. (A consumer ranging over the output
//     must exit cleanly. Close too early and you lose values or panic;
//     never close and the consumer hangs — ex2's bug again.)
//  3. Merge itself returns immediately; the shuttling happens in goroutines.
//  4. No leaks: when the inputs finish, every goroutine you started exits.
//  5. Race-free.
//
// This needs BOTH modules: one goroutine per input draining it into the
// output (range handles "until that input closes"), and the multi-sender
// close pattern from NOTES Section 4 — N senders on one channel, so who is
// allowed to close it, and what do they need to know first?
//
// Forbidden: time.Sleep, busy-wait polling with default.

// Merge fans-in every value from all inputs onto one output channel, which
// closes only when every input has closed.
func Merge(inputs ...<-chan int) <-chan int {
	panic("implement me")
}
