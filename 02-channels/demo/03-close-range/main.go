// Demo 03: close, buffer draining, comma-ok, and the panics.
//
//	go run ./02-channels/demo/03-close-range
//
// Shows, in order:
//  1. range over a channel ends exactly when the sender closes it
//  2. a closed channel first DRAINS its buffer (ok=true), then hands out
//     zero values with ok=false — immediately, forever, no blocking
//  3. send on a closed channel = panic (recovered here so you can read it)
package main

import "fmt"

func main() {
	// --- 1. range ends on close ---
	ch := make(chan string)
	go func() {
		for _, word := range []string{"pickup", "departure", "arrival"} {
			ch <- word
		}
		close(ch) // without this line, the range below never exits
	}()
	for w := range ch {
		fmt.Println("range got:", w)
	}
	fmt.Println("range exited — because the channel was closed")

	// --- 2. closed channels drain their buffer first ---
	buf := make(chan int, 3)
	buf <- 1
	buf <- 2
	close(buf) // two values still parked inside

	for i := 0; i < 4; i++ {
		v, ok := <-buf
		fmt.Printf("receive #%d: v=%d ok=%v\n", i+1, v, ok)
	}
	// #1 and #2 drain the buffer (ok=true); #3 and #4 return instantly:
	// zero value, ok=false. A closed channel never blocks a receiver.

	// --- 3. send on closed = panic ---
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("recovered from:", r)
			}
		}()
		closed := make(chan int)
		close(closed)
		closed <- 1 // panic: send on closed channel
	}()
	fmt.Println("still alive (that panic was recovered for demo purposes)")
}
