package exercises

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestFirstResponse_FastestWins(t *testing.T) {
	replicas := []string{"us-east", "eu-west", "ap-south"}
	query := func(r string) string {
		switch r {
		case "eu-west":
			time.Sleep(10 * time.Millisecond) // the fast one
		default:
			time.Sleep(200 * time.Millisecond)
		}
		return "answer from " + r
	}

	start := time.Now()
	got, err := FirstResponse(replicas, query, 1*time.Second)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "answer from eu-west" {
		t.Fatalf("got %q, want the fastest replica's answer", got)
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("took %v — must return when the FIRST answer arrives, not wait for stragglers", elapsed)
	}
}

func TestFirstResponse_Timeout(t *testing.T) {
	replicas := []string{"r1", "r2"}
	query := func(r string) string {
		time.Sleep(500 * time.Millisecond) // everyone is too slow
		return "late answer from " + r
	}

	start := time.Now()
	got, err := FirstResponse(replicas, query, 100*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("got %q with nil error — expected a timeout error", got)
	}
	if got != "" {
		t.Fatalf("got %q on timeout, want empty string", got)
	}
	if elapsed > 300*time.Millisecond {
		t.Fatalf("took %v — the timeout was 100ms; it must not wait for the slow replicas", elapsed)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "time") {
		t.Logf("note: error is %q — consider mentioning the timeout in it", err)
	}
}

func TestFirstResponse_NoLeaks(t *testing.T) {
	replicas := []string{"r1", "r2", "r3", "r4"}
	query := func(r string) string {
		time.Sleep(20 * time.Millisecond)
		return r
	}

	before := runtime.NumGoroutine()

	for i := 0; i < 25; i++ {
		_, _ = FirstResponse(replicas, query, 1*time.Second) // normal path
	}
	slow := func(r string) string { time.Sleep(80 * time.Millisecond); return r }
	for i := 0; i < 25; i++ {
		_, _ = FirstResponse(replicas, slow, 10*time.Millisecond) // timeout path
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before+2 {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("goroutines: %d before, %d after settle — the losing/late replicas are stuck delivering to nobody",
		before, runtime.NumGoroutine())
}
