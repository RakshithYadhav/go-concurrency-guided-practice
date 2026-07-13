package exercises

import (
	"context"
	"sync"
	"testing"
	"time"
)

// callRecorder timestamps every fetch call.
type callRecorder struct {
	mu    sync.Mutex
	times []time.Time
}

func (c *callRecorder) fetch(u string) string {
	c.mu.Lock()
	c.times = append(c.times, time.Now())
	c.mu.Unlock()
	return "res-" + u
}

func TestFetchAllPaced_ResultsInOrder(t *testing.T) {
	urls := []string{"a", "b", "c", "d", "e"}
	var rec callRecorder
	got, err := FetchAllPaced(context.Background(), urls, 1000, rec.fetch)
	if err != nil {
		t.Fatalf("want nil error, got %v", err)
	}
	if len(got) != len(urls) {
		t.Fatalf("got %d results, want %d", len(got), len(urls))
	}
	for i, u := range urls {
		if got[i] != "res-"+u {
			t.Fatalf("got[%d] = %q, want %q — input order required", i, got[i], "res-"+u)
		}
	}
}

func TestFetchAllPaced_Empty(t *testing.T) {
	got, err := FetchAllPaced(context.Background(), nil, 10, func(u string) string { return u })
	if err != nil || len(got) != 0 {
		t.Fatalf("want ([], nil), got (%v, %v)", got, err)
	}
}

func TestFetchAllPaced_ActuallyPaces(t *testing.T) {
	// 6 urls at 20/sec = 50ms between calls ≈ at least ~250ms total for
	// the 5 gaps. Unpaced code finishes in ~0ms and fails both checks.
	urls := []string{"a", "b", "c", "d", "e", "f"}
	var rec callRecorder

	start := time.Now()
	_, err := FetchAllPaced(context.Background(), urls, 20, rec.fetch)
	took := time.Since(start)
	if err != nil {
		t.Fatalf("want nil error, got %v", err)
	}

	if took < 200*time.Millisecond {
		t.Fatalf("6 calls at 20/sec finished in %v — that's not paced", took)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	for i := 1; i < len(rec.times); i++ {
		gap := rec.times[i].Sub(rec.times[i-1])
		if gap < 30*time.Millisecond { // 50ms nominal, generous tolerance
			t.Fatalf("gap between call %d and %d was %v — calls must be ~50ms apart", i-1, i, gap)
		}
	}
}

func TestFetchAllPaced_CancelStopsWaiting(t *testing.T) {
	// 100 urls at 5/sec would take ~20 seconds. Cancel at 100ms: the
	// function must let go promptly, not keep waiting for tokens.
	urls := make([]string, 100)
	for i := range urls {
		urls[i] = "u"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var rec callRecorder
	start := time.Now()
	got, err := FetchAllPaced(ctx, urls, 5, rec.fetch)
	took := time.Since(start)

	if err == nil {
		t.Fatalf("canceled at 100ms: want ctx error, got nil (%d results)", len(got))
	}
	if took > 600*time.Millisecond {
		t.Fatalf("took %v to return after a 100ms deadline — the token wait ignores ctx", took)
	}
	if len(got) >= len(urls) {
		t.Fatal("returned all results despite cancellation")
	}
}
