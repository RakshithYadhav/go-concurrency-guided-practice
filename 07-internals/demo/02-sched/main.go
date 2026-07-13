// Demo: the scheduler from NOTES Section 2, observable.
//
// Two experiments:
//   1. Async preemption — spinning goroutines with NO function calls
//      in the loop cannot starve a heartbeat goroutine (since Go 1.14).
//   2. Parked goroutines are cheap — 10,000 goroutines blocked on a
//      channel, and the stack memory it actually cost, measured.
//
// Run: go run ./07-internals/demo/02-sched
//
// Then run it again watching the scheduler itself (PowerShell):
//   $env:GODEBUG='schedtrace=1000'; go run ./07-internals/demo/02-sched; Remove-Item Env:GODEBUG
// Each line printed to stderr once a second shows:
//   runqueue=N   -> goroutines in the GLOBAL queue
//   [n n n ...]  -> each P's LOCAL queue length (one number per desk)
package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

var sink uint64 // spinners publish here so the loop can't be optimized away

func main() {
	fmt.Printf("machine: NumCPU=%d GOMAXPROCS=%d — that many desks (Ps)\n\n",
		runtime.NumCPU(), runtime.GOMAXPROCS(0))

	preemption()
	parkedAreCheap()
}

// Fill EVERY desk with a pure spin loop — no function calls, no channel
// ops, nothing the scheduler could hook cooperatively. Before Go 1.14
// the heartbeat below would freeze until the spinners finished. Today
// sysmon spots any goroutine running >10ms and preempts it, so the
// heartbeat keeps beating.
func preemption() {
	fmt.Println("=== 1. Async preemption: spin loops can't starve the heartbeat ===")
	spinners := runtime.GOMAXPROCS(0) // one per desk — all Ps busy
	var wg sync.WaitGroup

	fmt.Printf("   launching %d spinners, each counting to 2,000,000,000 (pure loop)\n", spinners)
	start := time.Now()
	for i := 0; i < spinners; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var x uint64
			for j := 0; j < 2_000_000_000; j++ {
				x++
			}
			sink += x // data published after the loop; the loop itself stays call-free
		}()
	}

	for beat := 1; beat <= 5; beat++ {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("   heartbeat %d at %v — still being scheduled despite %d busy desks\n",
			beat, time.Since(start).Round(time.Millisecond), spinners)
	}

	wg.Wait()
	fmt.Printf("   spinners finished in %v\n", time.Since(start).Round(time.Millisecond))
	fmt.Println("   (pre-1.14, those heartbeats would have printed only AFTER the spinners)")
	fmt.Println()
}

// Park 10,000 goroutines on one channel. Each is just a card in the
// channel's recvq — no OS thread attached. Measure what the stacks
// actually cost.
func parkedAreCheap() {
	fmt.Println("=== 2. 10,000 parked goroutines: cards in a folder, not threads ===")

	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	fmt.Printf("   before: %d goroutines, stack memory %.1f MB\n",
		runtime.NumGoroutine(), mb(before.StackSys))

	release := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 10_000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-release // park in recvq and stay there
		}()
	}
	time.Sleep(200 * time.Millisecond) // let them all park

	var during runtime.MemStats
	runtime.ReadMemStats(&during)
	fmt.Printf("   parked: %d goroutines, stack memory %.1f MB (+%.1f MB, ~%.1f KB each)\n",
		runtime.NumGoroutine(), mb(during.StackSys),
		mb(during.StackSys-before.StackSys),
		float64(during.StackSys-before.StackSys)/10_000/1024)

	fmt.Println("   releasing them (one close wakes all 10,000 — Section 1's broadcast)...")
	start := time.Now()
	close(release)
	wg.Wait()
	fmt.Printf("   all done in %v. 10k goroutines: trivial. 10k THREADS would not be.\n",
		time.Since(start).Round(time.Millisecond))
}

func mb(b uint64) float64 { return float64(b) / (1 << 20) }
