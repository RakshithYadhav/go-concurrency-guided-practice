package exercises

import (
	"fmt"
	"testing"
)

func TestProcessAll(t *testing.T) {
	jobs := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta", "eta", "theta"}
	process := func(job string) string { return "done:" + job }

	// Hammer it: the buggy version's goroutines usually all run after the
	// loop finished, so most slots get the LAST job's value — duplicates in
	// the wrong slots, correct values missing.
	for iter := 0; iter < 300; iter++ {
		results := ProcessAll(jobs, process)

		if len(results) != len(jobs) {
			t.Fatalf("iter %d: got %d results, want %d", iter, len(results), len(jobs))
		}
		for i, job := range jobs {
			want := fmt.Sprintf("done:%s", job)
			if results[i] != want {
				t.Fatalf("iter %d: results[%d] = %q, want %q — which job did this goroutine actually see?",
					iter, i, results[i], want)
			}
		}
	}
}
