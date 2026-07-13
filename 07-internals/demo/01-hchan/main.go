// Demo: the hchan machinery from NOTES Section 1, live.
//
// Every channel is one struct: a ring buffer, two wait queues
// (sendq/recvq), one lock. Each experiment below exercises one
// specific path through that machine and narrates it.
//
// Run: go run ./07-internals/demo/01-hchan
package main

import (
	"fmt"
	"time"
)

func main() {
	directHandoff()
	bufferThenPark()
	fifoWaiters()
	closeBroadcast()
	closedDrain()
}

// Send path 1: a receiver is ALREADY parked in recvq. The sender's
// value is copied straight to the receiver's stack — the buffer (there
// isn't one here anyway) is never touched.
func directHandoff() {
	fmt.Println("=== 1. Direct handoff: receiver waits first (send path 1) ===")
	ch := make(chan int) // unbuffered: no display case at all

	go func() {
		fmt.Println("   receiver: parking in recvq (nothing to take yet)...")
		v := <-ch
		fmt.Printf("   receiver: woke with %d — copied straight to my stack\n", v)
	}()

	time.Sleep(300 * time.Millisecond) // let the receiver park first
	fmt.Println("   sender:   recvq has a waiter -> handing off directly")
	start := time.Now()
	ch <- 42
	fmt.Printf("   sender:   send returned in %v (no blocking — the waiter was there)\n\n",
		time.Since(start).Round(time.Millisecond))
	time.Sleep(100 * time.Millisecond)
}

// Send paths 2 and 3, and the NOTES Section 1 numeric walkthrough:
// buffer fills (path 2), a third send parks (path 3), and a receive
// frees a slot AND moves the parked sender's value in — FIFO intact.
func bufferThenPark() {
	fmt.Println("=== 2. Buffered: fill, park, and the slot hand-me-down (paths 2+3) ===")
	ch := make(chan int, 2)

	ch <- 10
	ch <- 20
	fmt.Printf("   sent 10, 20 instantly — len=%d cap=%d (buffer full)\n", len(ch), cap(ch))

	parked := make(chan struct{})
	go func() {
		fmt.Println("   sender:   ch <- 30 ... no room, parking in sendq")
		ch <- 30
		close(parked)
	}()

	time.Sleep(300 * time.Millisecond)
	select {
	case <-parked:
		fmt.Println("   (unexpected: the send completed with a full buffer?)")
	default:
		fmt.Println("   main:     300ms later the sender is still parked. That's path 3.")
	}

	fmt.Printf("   main:     <-ch -> %d  (oldest out; parked sender's 30 moves into the freed slot)\n", <-ch)
	<-parked
	fmt.Printf("   main:     <-ch -> %d, <-ch -> %d  (FIFO survived: 10, 20, 30)\n\n", <-ch, <-ch)
}

// recvq is a FIFO queue: the receiver that parked FIRST is woken by
// the FIRST send.
func fifoWaiters() {
	fmt.Println("=== 3. recvq is FIFO: first parked, first served ===")
	ch := make(chan int)

	for i := 1; i <= 3; i++ {
		go func() {
			v := <-ch
			fmt.Printf("   receiver R%d: got %d\n", i, v)
		}()
		time.Sleep(100 * time.Millisecond) // park R1, then R2, then R3 — order known
	}

	for v := 1; v <= 3; v++ {
		ch <- v
		time.Sleep(50 * time.Millisecond) // keep the printout readable
	}
	fmt.Println("   (R1 parked first -> R1 got the first value, and so on)")
	fmt.Println()
}

// close() wakes EVERY parked receiver at once — each gets the zero
// value and ok=false. One close, many wake-ups: that's why close works
// as a broadcast (the done-channel trick from Modules 4 and 5).
func closeBroadcast() {
	fmt.Println("=== 4. close() is a broadcast: all of recvq wakes at once ===")
	ch := make(chan int)
	done := make(chan struct{})

	for i := 1; i <= 3; i++ {
		go func() {
			v, ok := <-ch
			fmt.Printf("   receiver R%d: woke with v=%d ok=%v\n", i, v, ok)
			done <- struct{}{}
		}()
	}

	time.Sleep(300 * time.Millisecond)
	fmt.Println("   main:     three receivers parked. Calling close(ch)...")
	close(ch)
	for i := 0; i < 3; i++ {
		<-done
	}
	fmt.Println()
}

// A closed buffered channel drains first: receives return the buffered
// items with ok=true, and only THEN the zero value with ok=false.
// Nothing ever blocks — the closed flag answers the question a
// receiver would have parked to wait for.
func closedDrain() {
	fmt.Println("=== 5. Closed + buffered: drain first, then zero values, never block ===")
	ch := make(chan int, 2)
	ch <- 10
	ch <- 20
	close(ch)

	for i := 1; i <= 3; i++ {
		start := time.Now()
		v, ok := <-ch
		fmt.Printf("   receive %d: v=%-2d ok=%-5v (returned in %v)\n",
			i, v, ok, time.Since(start).Round(time.Microsecond))
	}
	fmt.Println("   (two real values, then closed+empty -> instant zero value)")
}
