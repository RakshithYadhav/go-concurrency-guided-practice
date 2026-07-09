// Demo 02: WithTimeout — the same work, one deadline it beats, one it doesn't.
//
//	go run ./05-context/demo/02-timeout
//
// slowWork takes 200ms. First we give it 500ms (it finishes; Err()=nil
// until we cancel). Then we give it 100ms (the deadline fires first;
// Err()=DeadlineExceeded). Note the select shape — it's Module 4's
// done-channel pattern with ctx.Done() in the done seat.
package main

import (
	"context"
	"fmt"
	"time"
)

// slowWork simulates a 200ms operation that respects cancellation.
func slowWork(ctx context.Context) error {
	select {
	case <-time.After(200 * time.Millisecond): // the "work"
		return nil
	case <-ctx.Done(): // someone said stop
		return ctx.Err()
	}
}

func run(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // ALWAYS — even though the timeout would fire on its own

	start := time.Now()
	err := slowWork(ctx)
	took := time.Since(start).Round(time.Millisecond)

	if err != nil {
		fmt.Printf("timeout %v: gave up after %v — %v\n", timeout, took, err)
		return
	}
	fmt.Printf("timeout %v: finished in %v — no error\n", timeout, took)
}

func main() {
	fmt.Println("the work itself always takes 200ms")
	run(500 * time.Millisecond) // work wins
	run(100 * time.Millisecond) // deadline wins

	// And the OTHER reason a context dies: someone cancels it by hand.
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	err := slowWork(ctx)
	fmt.Printf("canceled by hand after 50ms — %v\n", err)
	fmt.Println("\nlesson: DeadlineExceeded = the clock won. Canceled = a person won.")
}
