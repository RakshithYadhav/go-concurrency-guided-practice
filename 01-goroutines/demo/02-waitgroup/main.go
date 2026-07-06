// Demo 02: sync.WaitGroup makes main provably wait.
//
//	go run ./01-goroutines/demo/02-waitgroup
//
// All 3 workers always print, but in a DIFFERENT ORDER across runs —
// completion is guaranteed, ordering is not. Both halves of that sentence
// matter.
//
// Also shown: the Go 1.25+ wg.Go shorthand, and why Add belongs OUTSIDE
// the goroutine.
package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	// Classic form. Add(1) happens synchronously, BEFORE the goroutine
	// starts. If Add ran inside the goroutine, Wait could observe the
	// counter at 0 and return before any worker even started.
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond) // pretend to work
			fmt.Println("classic worker", i, "done")
		}()
	}
	wg.Wait()
	fmt.Println("--- all classic workers finished ---")

	// Go 1.25+ shorthand: wg.Go does Add(1) + go + defer Done for you.
	var wg2 sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg2.Go(func() {
			fmt.Println("wg.Go worker", i, "done")
		})
	}
	wg2.Wait()
	fmt.Println("--- all wg.Go workers finished ---")
}
