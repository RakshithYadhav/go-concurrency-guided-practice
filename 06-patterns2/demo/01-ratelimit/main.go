// Demo 01: rate limiting — bounding speed, not just concurrency.
//
//	go run ./06-patterns2/demo/01-ratelimit
//
// The same 10 calls twice. Unlimited: all fire within a millisecond —
// this is what your "bounded" worker pool does to a downstream API.
// Limited to 20/sec (one token every 50ms, burst of 1): the timestamps
// spread out to ~50ms apart. The average speed is capped no matter how
// fast the callers are.
package main

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
)

func main() {
	fmt.Println("unlimited — 10 calls, timestamps relative to start:")
	start := time.Now()
	for i := 1; i <= 10; i++ {
		fmt.Printf("  call %2d at %6s\n", i, time.Since(start).Round(time.Millisecond))
	}

	fmt.Println("\nlimited to 20/sec (burst 1) — same 10 calls:")
	limiter := rate.NewLimiter(rate.Limit(20), 1)
	start = time.Now()
	for i := 1; i <= 10; i++ {
		if err := limiter.Wait(context.Background()); err != nil {
			fmt.Println("limiter wait:", err)
			return
		}
		fmt.Printf("  call %2d at %6s\n", i, time.Since(start).Round(time.Millisecond))
	}
	fmt.Println("\nlesson: concurrency limits protect YOUR machine;")
	fmt.Println("rate limits protect the machine you're calling.")
}
