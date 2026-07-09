package exercises

// Exercise 1 — IMPLEMENT: three pipeline stages that compose.
//
// Build the standard stage shape three times (NOTES Section 1):
//
//	Generate(nums...) — emits each number, then closes its output.
//	Square(in)        — reads until in closes, emits v*v, closes output.
//	Keep(in, pred)    — reads until in closes, emits only values where
//	                    pred(v) is true, closes output. (A filter stage:
//	                    not every input produces an output.)
//
// The tests compose them: Keep(Square(Generate(...)), isEven) must flow
// end-to-end and shut down cleanly. The timeout tests fail (not hang) if
// a stage forgets to close — remember the two rules of pipeline hygiene.
//
// Do not change signatures.

// ORIGINAL (before fix): ranged over the slice like a channel, sending
// indices instead of values —
//
//	for num := range nums {
//		out <- num   // sent 0,1,2,3,4 instead of 3,1,4,1,5
//	}
//
// Slice range with one variable gives the INDEX; channel range with one
// variable gives the VALUE. Fixed with `for _, num := range nums`.

// Generate returns a channel that emits each of nums, then closes.
func Generate(nums ...int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)
		for _, num := range nums {
			out <- num
		}

	}()

	return out
}

// Square returns a channel of v*v for every v received from in.
func Square(in <-chan int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)
		for num := range in {
			out <- num * num
		}

	}()

	return out
}

// Keep returns a channel that passes through only values where pred(v)
// is true.
func Keep(in <-chan int, pred func(int) bool) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)
		for num := range in {
			if pred(num) {
				out <- num
			}
		}

	}()

	return out
}
