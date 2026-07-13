package exercises

import (
	"runtime"
	"testing"
	"time"
)

func collectBatches(t *testing.T, out <-chan []int, within time.Duration) [][]int {
	t.Helper()
	var got [][]int
	deadline := time.After(within)
	for {
		select {
		case b, ok := <-out:
			if !ok {
				return got
			}
			if len(b) == 0 {
				t.Fatal("received an EMPTY batch — empty batches must never be sent")
			}
			got = append(got, b)
		case <-deadline:
			t.Fatalf("out never closed — missing close-flush? got so far: %v", got)
		}
	}
}

func TestBatch_SizeTrigger(t *testing.T) {
	in := make(chan int)
	out := Batch(in, 3, 5*time.Second) // age trigger effectively off

	go func() {
		defer close(in)
		for i := 1; i <= 7; i++ {
			in <- i
		}
	}()

	got := collectBatches(t, out, 3*time.Second)
	want := [][]int{{1, 2, 3}, {4, 5, 6}, {7}}
	if len(got) != len(want) {
		t.Fatalf("got %d batches %v, want %v", len(got), got, want)
	}
	for i := range want {
		if len(got[i]) != len(want[i]) {
			t.Fatalf("batch %d = %v, want %v", i, got[i], want[i])
		}
		for j := range want[i] {
			if got[i][j] != want[i][j] {
				t.Fatalf("batch %d = %v, want %v (order matters)", i, got[i], want[i])
			}
		}
	}
}

func TestBatch_AgeTrigger(t *testing.T) {
	in := make(chan int)
	out := Batch(in, 100, 80*time.Millisecond) // size trigger effectively off

	start := time.Now()
	go func() {
		in <- 1
		in <- 2
		// go quiet — the age trigger must flush {1,2} at ~80ms
		time.Sleep(300 * time.Millisecond)
		close(in)
	}()

	select {
	case b := <-out:
		took := time.Since(start)
		if len(b) != 2 || b[0] != 1 || b[1] != 2 {
			t.Fatalf("first batch = %v, want [1 2]", b)
		}
		if took < 50*time.Millisecond {
			t.Fatalf("batch arrived at %v — flushed before the 80ms age was reached?", took)
		}
		if took > 250*time.Millisecond {
			t.Fatalf("batch arrived at %v — the age trigger (80ms) never fired; only close flushed it", took)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no batch ever arrived")
	}
	collectBatches(t, out, 2*time.Second) // drain to close
}

func TestBatch_CloseFlushesPartial(t *testing.T) {
	in := make(chan int)
	out := Batch(in, 10, 5*time.Second)

	go func() {
		in <- 42
		in <- 43
		close(in) // neither size (10) nor age (5s) fired — close must flush
	}()

	got := collectBatches(t, out, 2*time.Second)
	if len(got) != 1 || len(got[0]) != 2 || got[0][0] != 42 || got[0][1] != 43 {
		t.Fatalf("got %v, want one batch [42 43] flushed on close", got)
	}
}

func TestBatch_EmptyInput(t *testing.T) {
	in := make(chan int)
	close(in)
	got := collectBatches(t, Batch(in, 5, 50*time.Millisecond), 2*time.Second)
	if len(got) != 0 {
		t.Fatalf("empty input: want no batches, got %v", got)
	}
}

func TestBatch_NoLeak(t *testing.T) {
	before := runtime.NumGoroutine()
	for i := 0; i < 20; i++ {
		in := make(chan int)
		out := Batch(in, 4, 30*time.Millisecond)
		go func() {
			for j := 0; j < 10; j++ {
				in <- j
			}
			close(in)
		}()
		for range out {
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		if runtime.NumGoroutine() <= before+3 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("goroutines before: %d, after 20 runs: %d — the batcher goroutine isn't exiting",
				before, runtime.NumGoroutine())
		}
		time.Sleep(20 * time.Millisecond)
	}
}
