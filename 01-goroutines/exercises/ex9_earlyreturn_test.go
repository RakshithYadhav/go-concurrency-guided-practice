package exercises

import (
	"fmt"
	"testing"
	"time"
)

// A real hang would freeze `go test` forever. To turn that into a clean
// failure instead, run BuildReports on its own goroutine and race it against
// a timeout with select — a useful pattern for testing anything that might
// deadlock.
func TestBuildReports(t *testing.T) {
	ids := []int{1, -2, 3, 4, -5, 6}
	build := func(id int) (string, error) {
		return fmt.Sprintf("report-%d", id), nil
	}

	done := make(chan []string, 1)
	go func() {
		done <- BuildReports(ids, build)
	}()

	select {
	case reports := <-done:
		if len(reports) != len(ids) {
			t.Fatalf("got %d reports, want %d", len(reports), len(ids))
		}
		for i, id := range ids {
			if id < 0 {
				if reports[i] != "" {
					t.Errorf("reports[%d] (invalid id %d) = %q, want empty", i, id, reports[i])
				}
				continue
			}
			want := fmt.Sprintf("report-%d", id)
			if reports[i] != want {
				t.Errorf("reports[%d] = %q, want %q", i, reports[i], want)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("BuildReports hung — some goroutine returned early without ever reaching wg.Done()")
	}
}

func TestBuildReports_AllValid(t *testing.T) {
	ids := []int{10, 20, 30}
	build := func(id int) (string, error) { return fmt.Sprintf("report-%d", id), nil }

	done := make(chan []string, 1)
	go func() { done <- BuildReports(ids, build) }()

	select {
	case reports := <-done:
		for i, id := range ids {
			want := fmt.Sprintf("report-%d", id)
			if reports[i] != want {
				t.Errorf("reports[%d] = %q, want %q", i, reports[i], want)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("BuildReports hung with no invalid ids at all")
	}
}
