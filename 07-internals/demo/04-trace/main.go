// Demo: a trace of an unbalanced pipeline (NOTES §4).
//
// The pipeline: generate -> enrich (4 workers, fast) -> commit
// (1 worker, SLOW — 20ms per item). The four enrich workers spend most
// of their lives parked, waiting for the slow committer to take their
// output. A CPU profile shows almost nothing — waiting burns no CPU.
// The TRACE shows it instantly.
//
// Run:  go run ./07-internals/demo/04-trace
// Then: go tool trace 07-internals/demo/04-trace/trace.out
//
// In the browser that opens:
//   - "Goroutine analysis" -> the enrich goroutines: look at how much
//     of their time is "Block (chan send)" vs "Running". That's
//     backpressure made visible.
//   - "View trace by proc" -> the timeline. Named regions show every
//     enrich (short marks, four lanes) and every commit (a solid
//     back-to-back train on one lane). The train IS the bottleneck.
package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime/trace"
	"sync"
	"time"
)

const items = 100

func main() {
	// Write trace.out next to this file, wherever the demo is run from.
	out := filepath.Join("07-internals", "demo", "04-trace", "trace.out")
	if _, err := os.Stat(filepath.Dir(out)); err != nil {
		out = "trace.out" // fallback: current directory
	}
	f, err := os.Create(out)
	if err != nil {
		fmt.Println("create trace file:", err)
		return
	}
	defer f.Close()

	if err := trace.Start(f); err != nil {
		fmt.Println("start trace:", err)
		return
	}
	start := time.Now()
	runPipeline()
	took := time.Since(start)
	trace.Stop()

	abs, _ := filepath.Abs(out)
	fmt.Printf("pipeline processed %d items in %v\n", items, took.Round(time.Millisecond))
	fmt.Println("\ntrace written to:", abs)
	fmt.Println("open it with:      go tool trace", out)
	fmt.Println("\nlook for: enrich workers parked in 'Block (chan send)' most of")
	fmt.Println("their lives, while commit runs a solid back-to-back train.")
	fmt.Println("One slow stage sets the pace of the whole line (Module 4's")
	fmt.Println("backpressure — now on a timeline).")
}

func runPipeline() {
	ctx := context.Background()
	gen := make(chan int)
	enriched := make(chan int)

	go func() {
		defer close(gen)
		for i := 0; i < items; i++ {
			gen <- i
		}
	}()

	// enrich: 4 workers, ~1ms of real CPU each — fast.
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for v := range gen {
				trace.WithRegion(ctx, "enrich", func() {
					buf := make([]byte, 32*1024)
					for i := 0; i < 20; i++ {
						sha256.Sum256(buf)
					}
				})
				enriched <- v
			}
		}()
	}
	go func() {
		wg.Wait()
		close(enriched)
	}()

	// commit: ONE worker, 20ms per item — the bottleneck.
	for range enriched {
		trace.WithRegion(ctx, "commit", func() {
			time.Sleep(20 * time.Millisecond)
		})
	}
}
