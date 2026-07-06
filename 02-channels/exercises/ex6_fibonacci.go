package exercises

// Exercise 6 — IMPLEMENT: a second producer, same concept, different shape.
//
// Fibonacci streams the first n Fibonacci numbers (0, 1, 1, 2, 3, 5, 8, ...),
// in order, then signals done. Deliberately the same core shape as ex1's
// Generate — if that pattern hasn't fully clicked yet, this is where it
// either clicks or reveals exactly what's still fuzzy.
//
// One structural difference from Generate, on purpose: every value here
// DEPENDS on the two values immediately before it. There is no way to
// produce Fibonacci number #7 without already having #5 and #6 in hand.
// Keep that in mind while deciding how many goroutines this needs, and
// whether "one goroutine per value" could even possibly work here.
//
// Requirements:
//  1. Return immediately — the sending happens in a goroutine you start,
//     NOT in Fibonacci itself. Same reason as Generate: the caller cannot
//     receive before Fibonacci returns the channel to them.
//  2. Deliver exactly n values, in order: 0, 1, 1, 2, 3, 5, 8, 13, ...
//  3. A consumer using `for v := range Fibonacci(n)` must exit the loop
//     after the nth value.
//  4. Fibonacci(0) must not hang a ranging consumer — the loop body simply
//     never runs.
//  5. The returned channel is receive-only.
//
// Forbidden:
//  - time.Sleep.
//  - Buffering sized to fit all n values upfront. Even less defensible here
//    than in ex1: precomputing every value before returning defeats the
//    entire point of STREAMING a sequence.
//  - More than one sending goroutine. Given each value needs the previous
//    two, what would concurrent, unordered goroutines even compute?

// Fibonacci streams the first n Fibonacci numbers, in order, then closes.
func Fibonacci(n int) <-chan int {
	panic("implement me")
}
