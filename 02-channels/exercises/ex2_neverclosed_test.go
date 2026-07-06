package exercises

import (
	"testing"
	"time"
)

func TestCollectSquares(t *testing.T) {
	done := make(chan []int, 1)
	go func() {
		done <- CollectSquares([]int{1, 2, 3, 4, 5})
	}()

	select {
	case got := <-done:
		want := []int{1, 4, 9, 16, 25}
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("got %v, want %v", got, want)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CollectSquares never returned — the range is waiting for something that never happens")
	}
}

func TestCollectSquares_Empty(t *testing.T) {
	done := make(chan []int, 1)
	go func() { done <- CollectSquares(nil) }()

	select {
	case got := <-done:
		if len(got) != 0 {
			t.Fatalf("got %v, want empty", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CollectSquares hung even with zero inputs")
	}
}
