package exercises

// Exercise 2 — FIX THE BUG: the range that never ends.
//
// CollectSquares streams squares through a channel and collects them. It
// computes every single value correctly. It also never returns. No panic,
// no error, no race report — just a function that hangs forever. This is
// THE most common channel bug in real code, and its symptom ("my function/
// service just stops making progress") gives you nothing to grep for.
//
// Your tasks:
//  1. Before touching anything, answer from the NOTES: what is the ONLY
//     thing that makes `for v := range ch` terminate? Now look at the
//     producer goroutine — does that thing ever happen?
//  2. Fix it with one line — in the right place. Wrong places compile too:
//     what goes wrong if you put it after the range loop instead? (Answer
//     via the axioms table, then convince yourself by trying it if unsure.)
//
// Do not change the function signature.

// CollectSquares returns the squares of nums, computed by a producer
// goroutine and streamed over a channel. BUG: it never returns.
func CollectSquares(nums []int) []int {
	squares := make(chan int)

	go func() {
		for _, n := range nums {
			squares <- n * n
		}
	}()
	close(squares)
	var results []int
	for sq := range squares {
		results = append(results, sq)
	}
	return results
}
