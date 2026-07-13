// Demo 04: batching — flush on size OR age, whichever fires first.
//
//	go run ./06-patterns2/demo/04-batch
//
// The producer sends a fast burst of 7 (watch size-triggered batches of
// 3), then goes quiet mid-batch (watch the age trigger flush a partial
// batch after 150ms), then sends a final trickle and closes (watch the
// close-flush).
package main

import (
	"fmt"
	"time"
)

func batch(in <-chan int, maxSize int, maxWait time.Duration) <-chan []int {
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
			fmt.Printf("  flush (%s): %v\n", reason, buf)
			out <- buf
			buf = nil
		}

		for {
			select {
			case v, ok := <-in:
				if !ok { // input closed: flush what's left, stop
					flush("close")
					return
				}
				if len(buf) == 0 {
					timer.Reset(maxWait) // clock starts with the batch's FIRST item
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

func main() {
	in := make(chan int)
	out := batch(in, 3, 150*time.Millisecond)

	go func() {
		defer close(in)
		for i := 1; i <= 7; i++ { // fast burst: two size-flushes + 1 leftover
			in <- i
		}
		time.Sleep(300 * time.Millisecond) // quiet: leftover flushes on age
		in <- 8                            // final trickle...
		in <- 9
	}() // ...then close: close-flush

	var count int
	for b := range out {
		count += len(b)
	}
	fmt.Printf("all %d items delivered, in order, in batches\n", count)
}
