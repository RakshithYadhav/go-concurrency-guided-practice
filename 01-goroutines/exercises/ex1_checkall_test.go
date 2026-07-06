package exercises

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestCheckAll_Correctness(t *testing.T) {
	targets := []string{"auth", "billing", "search", "mail", "cdn"}
	boom := errors.New("connection refused")

	check := func(target string) error {
		time.Sleep(10 * time.Millisecond)
		if target == "billing" || target == "cdn" {
			return boom
		}
		return nil
	}

	results := CheckAll(targets, check)

	if len(results) != len(targets) {
		t.Fatalf("got %d results, want %d", len(results), len(targets))
	}
	for i, r := range results {
		if r.Target != targets[i] {
			t.Errorf("results[%d].Target = %q, want %q (order must match input)", i, r.Target, targets[i])
		}
		wantErr := targets[i] == "billing" || targets[i] == "cdn"
		if (r.Err != nil) != wantErr {
			t.Errorf("results[%d] (%s): err = %v, wantErr = %v", i, targets[i], r.Err, wantErr)
		}
	}
}

func TestCheckAll_Empty(t *testing.T) {
	results := CheckAll(nil, func(string) error { return nil })
	if len(results) != 0 {
		t.Fatalf("got %d results for nil targets, want 0", len(results))
	}
}

// Proves the checks actually run concurrently: 8 checks x 100ms each is 800ms
// serially. Concurrent execution must land well under half of that.
func TestCheckAll_IsConcurrent(t *testing.T) {
	var targets []string
	for i := 0; i < 8; i++ {
		targets = append(targets, fmt.Sprintf("svc-%d", i))
	}
	check := func(string) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	}

	start := time.Now()
	results := CheckAll(targets, check)
	elapsed := time.Since(start)

	if len(results) != len(targets) {
		t.Fatalf("got %d results, want %d", len(results), len(targets))
	}
	if elapsed > 400*time.Millisecond {
		t.Fatalf("took %v for 8 x 100ms checks — that's (nearly) serial, not concurrent", elapsed)
	}
}
