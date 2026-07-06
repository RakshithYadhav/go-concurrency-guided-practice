package exercises

import (
	"fmt"
	"testing"
	"time"
)

// Both subtests must pass with the SAME implementation. Any fixed sleep S
// fails one of them: slow sends need S >= 300ms to complete, fast sends need
// S <= 120ms to be quick. Only real synchronization satisfies both.
func TestNotifyAll(t *testing.T) {
	devices := []string{"phone-1", "phone-2", "tablet-1", "watch-1", "laptop-1"}

	cases := []struct {
		name       string
		sendDelay  time.Duration
		maxElapsed time.Duration
	}{
		{"slow sends must all complete", 300 * time.Millisecond, 450 * time.Millisecond},
		{"fast sends must return promptly", 20 * time.Millisecond, 120 * time.Millisecond},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			send := func(device, msg string) string {
				time.Sleep(tc.sendDelay)
				return fmt.Sprintf("delivered %q to %s", msg, device)
			}

			start := time.Now()
			receipts := NotifyAll(devices, "server maintenance at 02:00", send)
			elapsed := time.Since(start)

			if len(receipts) != len(devices) {
				t.Fatalf("got %d receipts, want %d", len(receipts), len(devices))
			}
			for i, d := range devices {
				if receipts[i] == "" {
					t.Errorf("receipts[%d] (%s) is empty — NotifyAll returned before the send finished", i, d)
				}
			}
			if elapsed > tc.maxElapsed {
				t.Errorf("took %v, limit %v — waiting on a guess instead of on the work?", elapsed, tc.maxElapsed)
			}
		})
	}
}
