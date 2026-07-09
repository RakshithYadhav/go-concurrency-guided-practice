package exercises

import (
	"sort"
	"sync/atomic"
	"testing"
	"time"
)

// concurrencyProbe wraps a job function and records how many calls run
// at the same moment, and the highest that number ever got.
type concurrencyProbe struct {
	current atomic.Int64
	peak    atomic.Int64
}

func (p *concurrencyProbe) wrap(fn func(int) int) func(int) int {
	return func(v int) int {
		cur := p.current.Add(1)
		for {
			peak := p.peak.Load()
			if cur <= peak || p.peak.CompareAndSwap(peak, cur) {
				break
			}
		}
		defer p.current.Add(-1)
		return fn(v)
	}
}

func TestProcessAll_Correctness(t *testing.T) {
	jobs := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	got := ProcessAll(jobs, 3, func(v int) int { return v * v })
	want := []int{1, 4, 9, 16, 25, 36, 49, 64, 81, 100}
	if len(got) != len(want) {
		t.Fatalf("got %d results, want %d: %v", len(got), len(want), got)
	}
	sort.Ints(got)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted results = %v, want %v", got, want)
		}
	}
}

func TestProcessAll_EmptyJobs(t *testing.T) {
	got := ProcessAll(nil, 4, func(v int) int { return v })
	if len(got) != 0 {
		t.Fatalf("no jobs: want empty result, got %v", got)
	}
}

func TestProcessAll_MoreWorkersThanJobs(t *testing.T) {
	got := ProcessAll([]int{7}, 8, func(v int) int { return v + 1 })
	if len(got) != 1 || got[0] != 8 {
		t.Fatalf("got %v, want [8]", got)
	}
}

func TestProcessAll_NeverExceedsWorkers(t *testing.T) {
	var probe concurrencyProbe
	fn := probe.wrap(func(v int) int {
		time.Sleep(5 * time.Millisecond)
		return v
	})
	ProcessAll(make([]int, 40), 3, fn)
	if peak := probe.peak.Load(); peak > 3 {
		t.Fatalf("peak concurrency %d exceeded the worker count 3 — one goroutine per job?", peak)
	}
}

func TestProcessAll_ActuallyParallel(t *testing.T) {
	var probe concurrencyProbe
	fn := probe.wrap(func(v int) int {
		time.Sleep(20 * time.Millisecond)
		return v
	})
	ProcessAll(make([]int, 12), 4, fn)
	if peak := probe.peak.Load(); peak < 2 {
		t.Fatalf("peak concurrency was %d — workers never overlapped; is the pool actually concurrent?", peak)
	}
}
