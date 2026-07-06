package exercises

// Exercise 5 — FIX THE BUG: who is allowed to close?
//
// StreamAll fans out one goroutine per source, all sending filenames into a
// single shared output channel. The author knew "the sender must close the
// channel" — but there are THREE senders here, and each one closes when IT
// finishes. Run the test: it dies with a panic. Read the panic message and
// match it to a cell of the axioms table.
//
// Two distinct crimes are possible in this code, depending on timing:
//   - the fastest source closes while the others are still sending
//     → "send on closed channel"
//   - two sources both reach their close call
//     → "close of closed channel"
// Either way, the root cause is the same rule: close exactly ONCE, by
// exactly ONE goroutine, and only when no sender can possibly send again.
//
// Your tasks:
//  1. Say precisely which goroutine should close `out`, and what it must
//     wait for before doing so. NOTES Section 4 has the exact pattern —
//     and it needs a tool from Module 1, not a channel trick.
//  2. Restructure accordingly. Every value from every source must still
//     arrive, and the output must still close (the test ranges over it).
//
// Do not change the function signature.

// StreamAll merges the filenames from every source onto one channel.
// BUG: every source goroutine closes the shared channel itself.
func StreamAll(sources [][]string) <-chan string {
	out := make(chan string)

	for _, src := range sources {
		go func() {
			for _, filename := range src {
				out <- filename
			}
			close(out) // BUG: three senders, three closes, zero coordination
		}()
	}

	return out
}
