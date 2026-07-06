// Demo 02: a buffered channel is a mailbox with N slots.
//
//	go run ./02-channels/demo/02-buffered
//
// Capacity 2: the first two sends return instantly (values parked in the
// buffer, no receiver anywhere). The THIRD send blocks — buffer full — until
// the receiver drains one slot. Compare the timestamps with demo 01.
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

	ch := make(chan int, 2) // buffered: 2 slots

	go func() {
		for i := 1; i <= 3; i++ {
			stamp("sender", fmt.Sprintf("sending %d", i))
			ch <- i
			stamp("sender", fmt.Sprintf("send %d completed", i))
		}
	}()

	stamp("main", "sleeping 2s — watch sends 1 and 2 fly, send 3 freeze")
	time.Sleep(2 * time.Second)

	for i := 0; i < 3; i++ {
		stamp("main", fmt.Sprintf("received %d", <-ch))
	}
	time.Sleep(100 * time.Millisecond)
}
