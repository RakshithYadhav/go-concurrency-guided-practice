// Demo 01: an unbuffered channel is a handshake, not a mailbox.
//
//	go run ./02-channels/demo/01-unbuffered
//
// The sender reaches `ch <- 42` almost immediately — then freezes. Watch the
// timestamps: the send only completes at the exact moment the receiver shows
// up, two seconds later. Zero storage; the value passes hand-to-hand.
package main

import (
	"fmt"
	"time"
)

func main() {
	start := time.Now()
	stamp := func(who, what string) {
		fmt.Printf("[%5.2fs] %-8s %s\n", time.Since(start).Seconds(), who, what)
	}

	ch := make(chan int) // unbuffered: no second argument

	go func() {
		stamp("sender", "about to send 42")
		ch <- 42 // blocks HERE until the receiver arrives
		stamp("sender", "send completed — a receiver must have taken it")
	}()

	stamp("main", "sleeping 2s before receiving (sender is frozen meanwhile)")
	time.Sleep(2 * time.Second)

	stamp("main", "receiving now")
	v := <-ch
	stamp("main", fmt.Sprintf("got %d", v))

	time.Sleep(100 * time.Millisecond) // let the sender print its last line
}
