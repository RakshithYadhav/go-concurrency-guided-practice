package exercises

import "time"

// Exercise 4 — IMPLEMENT: a batcher — flush on size OR age.
//
// Batch reads items from `in` and groups them into slices on `out`,
// flushing whenever EITHER trigger fires (NOTES Section 5):
//
//   - SIZE: the batch reaches maxSize items → flush immediately
//   - AGE: the batch's OLDEST item has waited maxWait → flush whatever
//     is there (the age clock starts when the FIRST item of a new batch
//     arrives — an empty buffer has no age)
//   - CLOSE: when `in` closes, flush the final partial batch (if any),
//     then close `out`. Close-ownership rules from Module 2 apply: this
//     function's goroutine is the only sender on out, so it closes it.
//
// Requirements:
//   - never send an empty batch
//   - items stay in arrival order, across and within batches
//   - one goroutine, one select loop, one timer (time.NewTimer +
//     Reset — the demo shows the shape for a different trigger mix)
//   - no goroutine leak once `in` is closed and `out` is drained
//
// Do not change the signature.

// Batch groups items from in into slices of up to maxSize, flushing
// early once the oldest buffered item has waited maxWait.
func Batch(in <-chan int, maxSize int, maxWait time.Duration) <-chan []int {
	out := make(chan []int)
	go func() {
		defer close(out)
		var buf []int
		timer := time.NewTimer(maxWait)
		defer timer.Stop()

		flush := func(reason string) {
			if len(buf) == 0 {
				return
			}

			out <- buf
			buf = nil
		}

		for {
			select {
			case v, ok := <-in:
				if !ok {
					flush("close")
					return
				}
				if len(buf) == 0 {
					timer.Reset(maxWait)
				}
				buf = append(buf, v)
				if len(buf) >= maxSize {
					flush("size")
				}
			case <-timer.C:
				flush("age")
				timer.Reset(maxWait)

			}
		}
	}()
	return out
}

// ORIGINAL (before fix):
//
// func Batch(in <-chan int, maxSize int, maxWait time.Duration) <-chan []int {
// 	panic("implement me")
// }
