package exercises

import (
	"fmt"
	"testing"
	"time"
)

func TestTotalSize_Correctness(t *testing.T) {
	sizes := map[string]int64{
		"a.db": 1000, "b.db": 200, "c.db": 30, "d.db": 4, "e.db": 50000,
	}
	paths := []string{"a.db", "b.db", "c.db", "d.db", "e.db"}

	sizeOf := func(p string) int64 {
		time.Sleep(5 * time.Millisecond)
		return sizes[p]
	}

	got := TotalSize(paths, sizeOf)
	if want := int64(51234); got != want {
		t.Fatalf("TotalSize = %d, want %d", got, want)
	}
}

func TestTotalSize_Empty(t *testing.T) {
	if got := TotalSize(nil, func(string) int64 { return 1 }); got != 0 {
		t.Fatalf("TotalSize(nil) = %d, want 0", got)
	}
}

// Under -race, a shared `total += ...` across goroutines gets reported even
// when the sum happens to come out right. Lots of paths = lots of contention.
func TestTotalSize_ManyPaths(t *testing.T) {
	var paths []string
	var want int64
	sizes := make(map[string]int64)
	for i := 0; i < 200; i++ {
		p := fmt.Sprintf("file-%d", i)
		paths = append(paths, p)
		sizes[p] = int64(i * 7)
		want += int64(i * 7)
	}

	for iter := 0; iter < 50; iter++ {
		if got := TotalSize(paths, func(p string) int64 { return sizes[p] }); got != want {
			t.Fatalf("iter %d: TotalSize = %d, want %d (lost updates — shared accumulator?)", iter, got, want)
		}
	}
}

func TestTotalSize_IsConcurrent(t *testing.T) {
	var paths []string
	for i := 0; i < 8; i++ {
		paths = append(paths, fmt.Sprintf("slow-%d", i))
	}
	sizeOf := func(string) int64 {
		time.Sleep(100 * time.Millisecond)
		return 1
	}

	start := time.Now()
	got := TotalSize(paths, sizeOf)
	elapsed := time.Since(start)

	if got != 8 {
		t.Fatalf("TotalSize = %d, want 8", got)
	}
	if elapsed > 400*time.Millisecond {
		t.Fatalf("took %v for 8 x 100ms lookups — that's (nearly) serial, not concurrent", elapsed)
	}
}
