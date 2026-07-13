package exercises

import (
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"
)

func TestLeakHunt_CorrectAnswer(t *testing.T) {
	items := make([]int, 200)
	for i := range items {
		items[i] = 1
	}
	items[3] = 1000 // enriched: 1,000,000 — the first (and only) big one

	if got := FindFirstBig(items, 500); got != 1_000_000 {
		t.Fatalf("want 1000000, got %d", got)
	}
	if got := FindFirstBig([]int{1, 2, 3}, 500); got != -1 {
		t.Fatalf("no big items: want -1, got %d", got)
	}
}

func TestLeakHunt_NoGoroutineGrowth(t *testing.T) {
	items := make([]int, 200)
	for i := range items {
		items[i] = 1
	}
	items[3] = 1000 // hit early — plenty of items still behind it

	before := runtime.NumGoroutine()

	for i := 0; i < 100; i++ {
		if got := FindFirstBig(items, 500); got != 1_000_000 {
			t.Fatalf("call %d: wrong answer %d — fix must not break correctness", i, got)
		}
	}

	time.Sleep(300 * time.Millisecond) // let well-behaved goroutines finish and exit
	after := runtime.NumGoroutine()
	growth := after - before

	if growth > 20 {
		t.Errorf("goroutines: %d before, %d after 100 calls (+%d) — one stage leaks a goroutine per call",
			before, after, growth)
		t.Log("goroutine profile below. Find the big group; its stack names the guilty stage and line (NOTES §3):")
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
	}
}
