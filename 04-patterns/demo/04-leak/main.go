// Demo 04: THE FAILURE MODE — a leaky pipeline, measured live, then fixed.
//
//	go run ./04-patterns/demo/04-leak
//
// firstMatchLeaky takes only the FIRST value from its producer and
// returns. If the producer has more matches to send, it blocks on an
// unbuffered send forever — a leaked goroutine. Call it 50 times and the
// goroutine count climbs by ~50 and NEVER comes back down.
//
// firstMatchFixed wraps every send in a select with a done channel. The
// consumer's defer close(done) tells the producer "I left" — the producer
// takes the done case and exits. Same 50 calls: count stays flat.
package main

import (
	"fmt"
	"runtime"
	"time"
)

var items = []string{"a", "match-1", "b", "match-2", "match-3", "c"}

func isMatch(s string) bool { return len(s) > 1 }

func firstMatchLeaky() string {
	out := make(chan string)
	go func() {
		defer close(out)
		for _, it := range items {
			if isMatch(it) {
				out <- it // second match: nobody will ever receive → leak
			}
		}
	}()
	return <-out
}

func firstMatchFixed() string {
	out := make(chan string)
	done := make(chan struct{})
	defer close(done) // broadcast "consumer is leaving" on every return
	go func() {
		defer close(out)
		for _, it := range items {
			if isMatch(it) {
				select {
				case out <- it:
				case <-done: // consumer left — exit instead of blocking
					return
				}
			}
		}
	}()
	return <-out
}

func measure(name string, f func() string) {
	before := runtime.NumGoroutine()
	for i := 0; i < 50; i++ {
		f()
	}
	time.Sleep(50 * time.Millisecond) // let finished goroutines actually exit
	after := runtime.NumGoroutine()
	fmt.Printf("%-16s goroutines before: %2d   after 50 calls: %2d\n", name, before, after)
}

func main() {
	measure("leaky:", firstMatchLeaky)
	measure("fixed:", firstMatchFixed)
	fmt.Println("\nthe leaked goroutines above are blocked on a send forever —")
	fmt.Println("Go never garbage-collects a blocked goroutine.")
}
