package exercises

import (
	"sort"
	"testing"
	"time"
)

// feed returns a channel carrying the given values, closed at the end —
// a known-good producer so this test only exercises YOUR Merge.
func feed(vals ...int) <-chan int {
	ch := make(chan int)
	go func() {
		for _, v := range vals {
			ch <- v
		}
		close(ch)
	}()
	return ch
}

func TestMerge_AllValuesArrive(t *testing.T) {
	out := Merge(feed(1, 2, 3), feed(10, 20), feed(100))

	got := collectWithTimeout(t, out, 3*time.Second) // fails if out never closes
	want := []int{1, 2, 3, 10, 20, 100}

	if len(got) != len(want) {
		t.Fatalf("got %d values %v, want %d values %v", len(got), got, len(want), want)
	}
	sort.Ints(got)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("after sorting got %v, want %v (lost or duplicated values)", got, want)
		}
	}
}

func TestMerge_NoInputs(t *testing.T) {
	got := collectWithTimeout(t, Merge(), 2*time.Second)
	if len(got) != 0 {
		t.Fatalf("got %v from zero inputs, want nothing (but the channel must still close)", got)
	}
}

func TestMerge_SingleInput(t *testing.T) {
	got := collectWithTimeout(t, Merge(feed(7, 8, 9)), 2*time.Second)
	if len(got) != 3 {
		t.Fatalf("got %v, want [7 8 9]", got)
	}
}

func TestMerge_ReturnsImmediately(t *testing.T) {
	slow := make(chan int) // nobody ever sends; Merge must still return instantly
	start := time.Now()
	out := Merge(slow)
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("Merge took %v to return — shuttling must happen in goroutines", elapsed)
	}
	close(slow)
	collectWithTimeout(t, out, 2*time.Second)
}
