package exercises

import (
	"sort"
	"testing"
	"time"
)

// drainWithTimeout collects everything from ch, failing the test instead
// of hanging forever if the channel never closes.
func drainWithTimeout(t *testing.T, ch <-chan int) []int {
	t.Helper()
	var out []int
	timeout := time.After(2 * time.Second)
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, v)
		case <-timeout:
			t.Fatalf("channel never closed — did a stage forget close(out)? got so far: %v", out)
		}
	}
}

func TestGenerate(t *testing.T) {
	got := drainWithTimeout(t, Generate(3, 1, 4, 1, 5))
	want := []int{3, 1, 4, 1, 5}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v (order must be preserved)", got, want)
		}
	}
}

func TestGenerate_Empty(t *testing.T) {
	if got := drainWithTimeout(t, Generate()); len(got) != 0 {
		t.Fatalf("Generate() with no args: want empty closed channel, got %v", got)
	}
}

func TestSquare(t *testing.T) {
	got := drainWithTimeout(t, Square(Generate(1, 2, 3, 4)))
	want := []int{1, 4, 9, 16}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestKeep(t *testing.T) {
	got := drainWithTimeout(t, Keep(Generate(1, 2, 3, 4, 5, 6), func(v int) bool { return v%2 == 0 }))
	want := []int{2, 4, 6}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestFullPipeline(t *testing.T) {
	// squares of 1..6, keeping only the even ones: 4, 16, 36
	got := drainWithTimeout(t, Keep(Square(Generate(1, 2, 3, 4, 5, 6)), func(v int) bool { return v%2 == 0 }))
	want := []int{4, 16, 36}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	sort.Ints(got)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
