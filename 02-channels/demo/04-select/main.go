// Demo 04: select — multiplex, randomness, default, timeout.
//
//	go run ./02-channels/demo/04-select
//
// Four mini-scenes:
//  1. select waits on two channels at once; whichever delivers first wins
//  2. two channels ready SIMULTANEOUSLY: select picks uniformly at random —
//     run the demo a few times and watch the counts hover near 50/50
//  3. default = non-blocking: act only if something is ready right now
//  4. the timeout pattern: result vs clock, whichever comes first
package main

import (
	"fmt"
	"time"
)

func main() {
	// --- 1. whichever channel delivers first wins ---
	fast := make(chan string)
	slow := make(chan string)
	go func() { time.Sleep(30 * time.Millisecond); fast <- "fast replica" }()
	go func() { time.Sleep(300 * time.Millisecond); slow <- "slow replica" }()

	select {
	case v := <-fast:
		fmt.Println("1) winner:", v)
	case v := <-slow:
		fmt.Println("1) winner:", v)
	}

	// --- 2. both ready at once: uniformly random choice ---
	a := make(chan int, 1)
	b := make(chan int, 1)
	counts := map[string]int{}
	for i := 0; i < 1000; i++ {
		a <- 1 // both buffered channels are ready
		b <- 1 // before select even looks
		select {
		case <-a:
			counts["a"]++
		case <-b:
			counts["b"]++
		}
		// drain the loser so the next iteration starts clean
		select {
		case <-a:
		case <-b:
		}
	}
	fmt.Printf("2) both-ready choice over 1000 rounds: a=%d b=%d (≈50/50, never order-based)\n",
		counts["a"], counts["b"])

	// --- 3. default: non-blocking check ---
	queue := make(chan string, 1)
	select {
	case job := <-queue:
		fmt.Println("3) got job:", job)
	default:
		fmt.Println("3) queue empty right now — moving on without waiting")
	}

	// --- 4. the timeout pattern ---
	result := make(chan string, 1)
	go func() {
		time.Sleep(500 * time.Millisecond) // pretend this is a slow API call
		result <- "the answer"
	}()

	select {
	case v := <-result:
		fmt.Println("4) result:", v)
	case <-time.After(200 * time.Millisecond):
		fmt.Println("4) timed out after 200ms — the 500ms call lost the race")
	}
}
