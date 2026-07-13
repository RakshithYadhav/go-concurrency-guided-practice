// Demo 03: retry with exponential backoff + jitter.
//
//	go run ./06-patterns2/demo/03-retry
//
// The op fails three times, then succeeds. Watch the gaps between
// attempts: they roughly double (100ms → 200ms → 400ms), and jitter
// makes each one a bit different from a clean doubling — run the demo
// twice and the timestamps won't match. That randomness is what keeps
// 10,000 clients from retrying in synchronized waves.
package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

func retry(ctx context.Context, attempts int, base time.Duration, op func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = op(); err == nil {
			return nil
		}
		if i == attempts-1 {
			break // last attempt failed — no point waiting after it
		}
		wait := base * (1 << i)                                  // 100, 200, 400, ...
		wait += time.Duration(rand.Int63n(int64(wait) / 2))      // + jitter: 0–50% extra
		fmt.Printf("  attempt %d failed, waiting %v\n", i+1, wait.Round(time.Millisecond))
		select {
		case <-time.After(wait):
		case <-ctx.Done(): // canceled mid-wait: stop retrying NOW
			return ctx.Err()
		}
	}
	return err
}

func main() {
	calls := 0
	flaky := func() error {
		calls++
		if calls <= 3 {
			return errors.New("transient network blip")
		}
		return nil
	}

	start := time.Now()
	err := retry(context.Background(), 5, 100*time.Millisecond, flaky)
	fmt.Printf("result after %d calls in %v: err=%v\n",
		calls, time.Since(start).Round(time.Millisecond), err)
	fmt.Println("\nrun me again — the waits will be slightly different (jitter).")
}
