package exercises

import (
	"fmt"
	"testing"
	"time"
)

func TestFetchAllBounded_OrderPreserved(t *testing.T) {
	urls := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	got := FetchAllBounded(urls, 3, func(u string) string { return "res-" + u })
	if len(got) != len(urls) {
		t.Fatalf("got %d results, want %d", len(got), len(urls))
	}
	for i, u := range urls {
		if got[i] != "res-"+u {
			t.Fatalf("got[%d] = %q, want %q — results must be in input order", i, got[i], "res-"+u)
		}
	}
}

func TestFetchAllBounded_Empty(t *testing.T) {
	got := FetchAllBounded(nil, 4, func(u string) string { return u })
	if len(got) != 0 {
		t.Fatalf("no urls: want empty, got %v", got)
	}
}

func TestFetchAllBounded_RespectsLimit(t *testing.T) {
	var probe concurrencyProbe
	fetchProbed := func(u string) string {
		cur := probe.current.Add(1)
		for {
			peak := probe.peak.Load()
			if cur <= peak || probe.peak.CompareAndSwap(peak, cur) {
				break
			}
		}
		defer probe.current.Add(-1)
		time.Sleep(5 * time.Millisecond)
		return u
	}

	urls := make([]string, 30)
	for i := range urls {
		urls[i] = fmt.Sprintf("u%d", i)
	}
	FetchAllBounded(urls, 4, fetchProbed)
	if peak := probe.peak.Load(); peak > 4 {
		t.Fatalf("peak concurrency %d exceeded limit 4 — is the semaphore actually acquired before fetching?", peak)
	}
}

func TestFetchAllBounded_ActuallyParallel(t *testing.T) {
	var probe concurrencyProbe
	fetchProbed := func(u string) string {
		cur := probe.current.Add(1)
		for {
			peak := probe.peak.Load()
			if cur <= peak || probe.peak.CompareAndSwap(peak, cur) {
				break
			}
		}
		defer probe.current.Add(-1)
		time.Sleep(20 * time.Millisecond)
		return u
	}

	urls := make([]string, 12)
	for i := range urls {
		urls[i] = fmt.Sprintf("u%d", i)
	}
	FetchAllBounded(urls, 4, fetchProbed)
	if peak := probe.peak.Load(); peak < 2 {
		t.Fatalf("peak concurrency was %d — fetches never overlapped; running them one at a time?", peak)
	}
}
