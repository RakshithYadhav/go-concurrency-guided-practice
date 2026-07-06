package exercises

import (
	"testing"
	"time"
)

// collectWithTimeout ranges over ch, failing the test if the range doesn't
// finish within the deadline (i.e. the channel was never closed).
func collectWithTimeout(t *testing.T, ch <-chan int, deadline time.Duration) []int {
	t.Helper()
	done := make(chan []int, 1)
	go func() {
		var got []int
		for v := range ch {
			got = append(got, v)
		}
		done <- got
	}()
	select {
	case got := <-done:
		return got
	case <-time.After(deadline):
		t.Fatal("range over the channel never finished — was it ever closed?")
		return nil
	}
}

func TestGenerate_ValuesInOrder(t *testing.T) {
	got := collectWithTimeout(t, Generate(3, 1, 4, 1, 5), 2*time.Second)
	want := []int{3, 1, 4, 1, 5}
	if len(got) != len(want) {
		t.Fatalf("got %v (%d values), want %v", got, len(got), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v (order must be preserved)", got, want)
		}
	}
}

func TestGenerate_Empty(t *testing.T) {
	got := collectWithTimeout(t, Generate(), 2*time.Second)
	if len(got) != 0 {
		t.Fatalf("got %v, want no values", got)
	}
}

func TestGenerate_ReturnsImmediately(t *testing.T) {
	start := time.Now()
	ch := Generate(make([]int, 10_000)...) // nobody receiving yet
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("Generate took %v to return — it must start a goroutine, not send inline", elapsed)
	}
	collectWithTimeout(t, ch, 2*time.Second)
}
