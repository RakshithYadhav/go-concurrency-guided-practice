package exercises

// Exercise 4 — FIX THE BUG: goroutine leak.
//
// FirstResult fans a query out to N replicas and returns the fastest answer —
// a real pattern (hedged requests). It returns the right answer every time.
// It also permanently leaks N-1 goroutines per call. In a server handling
// thousands of requests, that's a slow-motion OOM.
//
// Your tasks:
//  1. Explain the leak: after the winner's value is received, what exactly
//     are the losing goroutines doing, and why can they NEVER finish?
//     (`results <- search(q)` on an unbuffered channel blocks until someone
//     receives — and nobody ever will again.)
//  2. Fix it with a ONE-LINE change. This previews Module 2 (channels), but
//     the fix itself needs no channel wizardry.
//  3. Golden rule check: after your fix, every goroutine this function starts
//     can always exit. Convince yourself before running the test.
//
// Do not change the function signature.

// FirstResult queries all replicas concurrently and returns whichever answers
// first. BUG: every call leaks len(queries)-1 goroutines, forever.
func FirstResult(queries []string, search func(query string) string) string {
	results := make(chan string)
	for _, q := range queries {
		go func() {
			results <- search(q)
		}()
	}
	return <-results
}
