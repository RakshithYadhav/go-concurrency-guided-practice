// Demo: a deliberately sick server to practice pprof on (NOTES §3).
//
// Two planted problems:
//   - a goroutine LEAK: every 50ms a leakyWorker starts and parks
//     forever on a channel nobody sends to (~20 leaked/second)
//   - a CPU HOTSPOT: burnCPU hashes in a tight loop, pegging one core
//
// Run it, then hunt both with pprof while it's alive:
//
//   go run ./07-internals/demo/03-pprof
//
//   The leak (watch it, then profile it):
//     browser:  http://localhost:6060/debug/pprof/            <- goroutine count climbing
//     browser:  http://localhost:6060/debug/pprof/goroutine?debug=1
//               -> the biggest group's stack says: chanrecv <- main.leakyWorker. Caught.
//     or:       go tool pprof -top http://localhost:6060/debug/pprof/goroutine
//
//   The hotspot (samples CPU for 10s, then shows the guilty function):
//     go tool pprof -top -seconds 10 http://localhost:6060/debug/pprof/profile
//               -> main.burnCPU at the top of the list. Caught.
//
// The server shuts itself down after -run (default 5m).
package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof" // registers /debug/pprof/* on the default mux
	"runtime"
	"time"
)

var never = make(chan struct{}) // no one ever sends; receivers park forever

// leakyWorker looks innocent — "wait for the result" — but the result
// channel is never written. Every call parks one goroutine forever in
// this exact spot. THIS is the name you'll see in the profile.
func leakyWorker() {
	<-never
}

// burnCPU is the hotspot: hashing the same buffer as fast as one core
// allows. In the CPU profile its name sits at the top.
func burnCPU() {
	buf := make([]byte, 64*1024)
	for {
		sha256.Sum256(buf)
	}
}

func main() {
	run := flag.Duration("run", 5*time.Minute, "how long to stay alive")
	flag.Parse()

	go func() {
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			fmt.Println("pprof server:", err)
		}
	}()

	go burnCPU()
	go func() {
		for {
			go leakyWorker()
			time.Sleep(50 * time.Millisecond) // ~20 leaks/second
		}
	}()

	fmt.Println("sick server is up — pprof at http://localhost:6060/debug/pprof/")
	fmt.Printf("running for %v (Ctrl+C to stop early)\n\n", *run)
	fmt.Println("watch the leak grow:")

	deadline := time.After(*run)
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			fmt.Printf("  goroutines now: %d (climbing ~20/s — a healthy server holds steady)\n",
				runtime.NumGoroutine())
		case <-deadline:
			fmt.Printf("\ntime's up. Final count: %d goroutines — every one of them\n",
				runtime.NumGoroutine())
			fmt.Println("parked in main.leakyWorker, exactly what the profile showed.")
			return
		}
	}
}
