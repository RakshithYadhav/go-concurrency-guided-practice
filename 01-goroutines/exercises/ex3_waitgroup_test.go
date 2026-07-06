package exercises

import (
	"fmt"
	"testing"
)

func TestFetchAll(t *testing.T) {
	ids := []int{10, 20, 30, 40, 50, 60, 70, 80}
	fetch := func(id int) string { return fmt.Sprintf("record-%d", id) }

	// The bug is timing-dependent, so hammer it: with Add() in the wrong
	// place, Wait() can return while the counter is still 0, before some
	// (or all) goroutines have run. Some iteration will come back with
	// missing entries — and -race flags the result writes racing the read.
	for iter := 0; iter < 500; iter++ {
		results := FetchAll(ids, fetch)

		if len(results) != len(ids) {
			t.Fatalf("iter %d: got %d results, want %d", iter, len(results), len(ids))
		}
		for i, id := range ids {
			want := fmt.Sprintf("record-%d", id)
			if results[i] != want {
				t.Fatalf("iter %d: results[%d] = %q, want %q — Wait() returned before the work finished",
					iter, i, results[i], want)
			}
		}
	}
}
