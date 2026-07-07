package exercises

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestMyOnce_RunsExactlyOnce(t *testing.T) {
	var (
		o     MyOnce
		calls atomic.Int32
		wg    sync.WaitGroup
	)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.Do(func() { calls.Add(1) })
		}()
	}
	wg.Wait()

	if n := calls.Load(); n != 1 {
		t.Fatalf("f ran %d times, want exactly 1", n)
	}
}

// Guarantee 2: a Do that "loses" must still block until the winner's f has
// COMPLETED. If Do returns early for losers, they observe initialized ==
// false while the winner is still sleeping inside f.
func TestMyOnce_LatecomersWaitForCompletion(t *testing.T) {
	var (
		o           MyOnce
		initialized atomic.Bool
		wg          sync.WaitGroup
	)

	slowInit := func() {
		time.Sleep(100 * time.Millisecond) // winner is in here a while
		initialized.Store(true)            // the very LAST thing init does
	}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.Do(slowInit)
			// Every Do return — winner or loser — must be AFTER slowInit
			// finished:
			if !initialized.Load() {
				t.Error("Do returned before f completed — guarantee 2 violated")
			}
		}()
	}
	wg.Wait()
}

func TestMyOnce_LaterCallsDoNotRunOtherFuncs(t *testing.T) {
	var (
		o      MyOnce
		first  atomic.Int32
		second atomic.Int32
	)
	o.Do(func() { first.Add(1) })
	o.Do(func() { second.Add(1) }) // different f — must NOT run
	o.Do(func() { first.Add(1) })  // same f as the winner — must not run again either

	if first.Load() != 1 || second.Load() != 0 {
		t.Fatalf("first ran %d times (want 1), second ran %d times (want 0)",
			first.Load(), second.Load())
	}
}

func TestMyOnce_SequentialCallsCheap(t *testing.T) {
	var o MyOnce
	o.Do(func() {})
	start := time.Now()
	for i := 0; i < 1_000_000; i++ {
		o.Do(func() { t.Error("must never run again") })
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("1M no-op Do calls took %v — after the first run, Do should be a fast check", elapsed)
	}
}
