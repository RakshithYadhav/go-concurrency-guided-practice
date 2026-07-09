package exercises

import (
	"context"
	"runtime"
	"sort"
	"testing"
	"time"
)

func TestProcessAllCtx_NoCancellation(t *testing.T) {
	jobs := []int{1, 2, 3, 4, 5, 6, 7, 8}
	got, err := ProcessAllCtx(context.Background(), jobs, 3, func(v int) int { return v * 10 })
	if err != nil {
		t.Fatalf("no cancellation: want nil error, got %v", err)
	}
	want := []int{10, 20, 30, 40, 50, 60, 70, 80}
	if len(got) != len(want) {
		t.Fatalf("got %d results %v, want %d", len(got), got, len(want))
	}
	sort.Ints(got)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted results = %v, want %v", got, want)
		}
	}
}

func TestProcessAllCtx_EmptyJobs(t *testing.T) {
	got, err := ProcessAllCtx(context.Background(), nil, 4, func(v int) int { return v })
	if err != nil || len(got) != 0 {
		t.Fatalf("want ([], nil), got (%v, %v)", got, err)
	}
}

func TestProcessAllCtx_StopsPromptlyOnCancel(t *testing.T) {
	// 200 jobs x 15ms at 4 workers ≈ 750ms if it ignores cancellation.
	// The deadline fires at 40ms; a listening pool returns well under 300ms.
	jobs := make([]int, 200)
	for i := range jobs {
		jobs[i] = i
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()

	start := time.Now()
	got, err := ProcessAllCtx(ctx, jobs, 4, func(v int) int {
		time.Sleep(15 * time.Millisecond)
		return v
	})
	took := time.Since(start)

	if err == nil {
		t.Fatalf("canceled mid-run: want ctx error, got nil (results: %d)", len(got))
	}
	if took > 300*time.Millisecond {
		t.Fatalf("pool took %v to let go after a 40ms deadline — it isn't listening to ctx", took)
	}
	if len(got) >= len(jobs) {
		t.Fatalf("got all %d results despite cancellation at 40ms — nothing stopped early", len(got))
	}
}

func TestProcessAllCtx_NoLeakAfterCancel(t *testing.T) {
	jobs := make([]int, 100)
	before := runtime.NumGoroutine()

	for i := 0; i < 20; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		ProcessAllCtx(ctx, jobs, 3, func(v int) int {
			time.Sleep(5 * time.Millisecond)
			return v
		})
		cancel()
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		if runtime.NumGoroutine() <= before+4 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("goroutines before: %d, after 20 canceled runs: %d — something is stuck on a send",
				before, runtime.NumGoroutine())
		}
		time.Sleep(20 * time.Millisecond)
	}
}
