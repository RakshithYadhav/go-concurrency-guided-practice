package exercises

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestFirstResult_Correctness(t *testing.T) {
	search := func(q string) string { return "answer:" + q }
	got := FirstResult([]string{"replica-a", "replica-b", "replica-c"}, search)
	if got != "answer:replica-a" && got != "answer:replica-b" && got != "answer:replica-c" {
		t.Fatalf("got %q, want an answer from one of the replicas", got)
	}
}

func TestFirstResult_NoLeak(t *testing.T) {
	queries := []string{"r1", "r2", "r3", "r4", "r5"}
	search := func(q string) string {
		time.Sleep(5 * time.Millisecond) // losers finish shortly after the winner
		return "answer:" + q
	}

	before := runtime.NumGoroutine()

	// 30 calls x 5 replicas: the buggy version accumulates ~120 stuck
	// goroutines. A fixed version returns to baseline once stragglers finish.
	for i := 0; i < 30; i++ {
		_ = FirstResult(queries, search)
	}

	// Give non-leaked stragglers time to finish and be reaped.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before+2 { // small tolerance for runtime noise
			return // no leak
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("goroutines: %d before, %d after 2s settle — the losers are stuck forever (leak). %s",
		before, runtime.NumGoroutine(),
		fmt.Sprintf("Each call leaks len(queries)-1 = %d goroutines.", len(queries)-1))
}
