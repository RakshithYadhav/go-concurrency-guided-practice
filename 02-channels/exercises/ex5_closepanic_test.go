package exercises

import (
	"sort"
	"testing"
	"time"
)

// Note: with the ORIGINAL bug, this test doesn't "fail" politely — the whole
// test binary crashes with a panic (send on closed channel / close of closed
// channel). That crash IS the expected starting point; read its message.
func TestStreamAll(t *testing.T) {
	sources := [][]string{
		{"a1.csv", "a2.csv", "a3.csv", "a4.csv"},
		{"b1.csv", "b2.csv"},
		{"c1.csv", "c2.csv", "c3.csv"},
	}

	done := make(chan []string, 1)
	go func() {
		var got []string
		for f := range StreamAll(sources) {
			got = append(got, f)
		}
		done <- got
	}()

	select {
	case got := <-done:
		want := []string{"a1.csv", "a2.csv", "a3.csv", "a4.csv", "b1.csv", "b2.csv", "c1.csv", "c2.csv", "c3.csv"}
		if len(got) != len(want) {
			t.Fatalf("got %d filenames %v, want %d — lost values (closed too early?)", len(got), got, len(want))
		}
		sort.Strings(got)
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("after sorting got %v, want %v", got, want)
			}
		}
	case <-time.After(3 * time.Second):
		t.Fatal("StreamAll's output never closed — did anyone end up responsible for closing it?")
	}
}

func TestStreamAll_Repeated(t *testing.T) {
	// The original bug is timing-dependent — hammer it so it can't hide.
	for iter := 0; iter < 100; iter++ {
		sources := [][]string{{"x", "y"}, {"z"}}
		done := make(chan int, 1)
		go func() {
			n := 0
			for range StreamAll(sources) {
				n++
			}
			done <- n
		}()
		select {
		case n := <-done:
			if n != 3 {
				t.Fatalf("iter %d: received %d values, want 3", iter, n)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("iter %d: output never closed", iter)
		}
	}
}
