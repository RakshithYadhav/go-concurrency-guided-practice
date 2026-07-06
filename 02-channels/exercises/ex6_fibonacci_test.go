package exercises

import (
	"testing"
	"time"
)

func TestFibonacci_ValuesInOrder(t *testing.T) {
	got := collectWithTimeout(t, Fibonacci(10), 2*time.Second)
	want := []int{0, 1, 1, 2, 3, 5, 8, 13, 21, 34}

	if len(got) != len(want) {
		t.Fatalf("got %v (%d values), want %v", got, len(got), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v (order/values must match exactly)", got, want)
		}
	}
}

func TestFibonacci_Zero(t *testing.T) {
	got := collectWithTimeout(t, Fibonacci(0), 2*time.Second)
	if len(got) != 0 {
		t.Fatalf("got %v, want no values", got)
	}
}

func TestFibonacci_One(t *testing.T) {
	got := collectWithTimeout(t, Fibonacci(1), 2*time.Second)
	want := []int{0}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFibonacci_ReturnsImmediately(t *testing.T) {
	start := time.Now()
	ch := Fibonacci(10_000) // nobody receiving yet — must not precompute all of these
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("Fibonacci took %v to return — it must start a goroutine, not compute inline", elapsed)
	}
	// Only drain a handful — proves values stream rather than requiring the
	// full n to be ready upfront. (Draining fully would also work, but this
	// keeps the test fast even for a large n.)
	for i := 0; i < 5; i++ {
		<-ch
	}
}

func TestFibonacci_RepeatedCallsAgree(t *testing.T) {
	// Same input, many calls: catches any hidden shared state between calls
	// (e.g. package-level variables reused across Fibonacci invocations).
	for iter := 0; iter < 20; iter++ {
		got := collectWithTimeout(t, Fibonacci(8), 2*time.Second)
		want := []int{0, 1, 1, 2, 3, 5, 8, 13}
		if len(got) != len(want) {
			t.Fatalf("iter %d: got %v, want %v", iter, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("iter %d: got %v, want %v", iter, got, want)
			}
		}
	}
}
