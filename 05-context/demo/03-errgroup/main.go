// Demo 03: errgroup — the first failure pulls the plug on everyone else.
//
//	go run ./05-context/demo/03-errgroup
//
// Five "fetches". Four would take a full second. One fails at 50ms.
// Because errgroup.WithContext cancels the group ctx on the first error,
// the four slow fetches stop at ~50ms instead of burning their second.
// Total wall time proves it — watch the clock.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

func fetch(ctx context.Context, name string, d time.Duration, fail bool) error {
	start := time.Now()
	if fail {
		time.Sleep(d)
		fmt.Printf("  %s: FAILED on purpose after %v\n", name, time.Since(start).Round(time.Millisecond))
		return errors.New(name + " exploded")
	}
	select {
	case <-time.After(d):
		fmt.Printf("  %s: finished full work in %v\n", name, time.Since(start).Round(time.Millisecond))
		return nil
	case <-ctx.Done():
		fmt.Printf("  %s: canceled after %v (%v)\n", name, time.Since(start).Round(time.Millisecond), context.Cause(ctx))
		return ctx.Err()
	}
}

func main() {
	g, ctx := errgroup.WithContext(context.Background())
	start := time.Now()

	g.Go(func() error { return fetch(ctx, "slow-1", time.Second, false) })
	g.Go(func() error { return fetch(ctx, "slow-2", time.Second, false) })
	g.Go(func() error { return fetch(ctx, "slow-3", time.Second, false) })
	g.Go(func() error { return fetch(ctx, "slow-4", time.Second, false) })
	g.Go(func() error { return fetch(ctx, "bad-egg", 50*time.Millisecond, true) })

	err := g.Wait()
	fmt.Printf("\ng.Wait() returned after %v with: %v\n", time.Since(start).Round(time.Millisecond), err)
	fmt.Println("lesson: without errgroup this takes 1s no matter what;")
	fmt.Println("with it, one failure at 50ms stops all five at ~50ms.")
}
