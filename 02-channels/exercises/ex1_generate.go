package exercises

// Exercise 1 — IMPLEMENT: your first producer.
//
// Generate returns a channel that delivers each of the given numbers, in
// order, then signals "no more" so consumers can stop cleanly. This is the
// canonical producer shape — every pipeline in Module 4 starts with a
// function that looks exactly like this.
//
// Requirements:
//  1. Return immediately — the sending happens in a goroutine you start,
//     NOT in Generate itself. (Think: what happens on an unbuffered channel
//     if you send before returning the channel to the caller? Who would be
//     receiving?)
//  2. Deliver every value of nums, in order.
//  3. A consumer using `for v := range Generate(...)` must EXIT the loop
//     after the last value. What does range need for that to happen, and
//     which goroutine is allowed to do it?
//  4. Generate(no arguments) must not hang a ranging consumer — the loop
//     body just never runs.
//  5. The returned channel is receive-only (<-chan int) — already in the
//     signature; your implementation works with a bidirectional channel
//     internally and returns it (the conversion is automatic).
//
// Forbidden: time.Sleep, buffering sized to "fit everything" so you can
// send without a goroutine (that dodges requirement 1's lesson).

// ORIGINAL (before fix) — kept for revision / re-attempting from scratch:
//
//	func Generate(nums ...int) <-chan int {
//		panic("implement me")
//	}

// Generate returns a channel that yields each of nums in order, then closes.
func Generate(nums ...int) <-chan int {
	ch := make(chan int)
	go func() {
		for _, num := range nums {
			ch <- num
		}
		close(ch)
	}()
	return ch
}
