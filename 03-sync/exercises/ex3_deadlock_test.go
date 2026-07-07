package exercises

import (
	"testing"
	"time"
)

// The failure mode is a permanent block, so every scenario runs behind the
// hang-guard pattern (goroutine + select + timeout) from Module 1 ex9.
func withTimeout(t *testing.T, name string, deadline time.Duration, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		f()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(deadline):
		t.Fatalf("%s: still blocked after %v — is a lock being held forever?", name, deadline)
	}
}

func TestReserve_HappyPath(t *testing.T) {
	inv := NewInventory(map[string]int{"bolts": 10})
	withTimeout(t, "two successful reserves", 2*time.Second, func() {
		if err := inv.Reserve("bolts", 4); err != nil {
			t.Errorf("first reserve: %v", err)
		}
		if err := inv.Reserve("bolts", 4); err != nil {
			t.Errorf("second reserve: %v", err)
		}
		if got := inv.Available("bolts"); got != 2 {
			t.Errorf("available = %d, want 2", got)
		}
	})
}

func TestReserve_FailureThenAnything(t *testing.T) {
	inv := NewInventory(map[string]int{"bolts": 3})

	withTimeout(t, "failed reserve then ANY later call", 2*time.Second, func() {
		if err := inv.Reserve("bolts", 999); err == nil {
			t.Error("expected an insufficient-stock error")
		}
		// The buggy version never gets past this line:
		if err := inv.Reserve("bolts", 1); err != nil {
			t.Errorf("small reserve after a failed one: %v", err)
		}
		if got := inv.Available("bolts"); got != 2 {
			t.Errorf("available = %d, want 2", got)
		}
	})
}

func TestReserve_Concurrent(t *testing.T) {
	inv := NewInventory(map[string]int{"widgets": 50})

	withTimeout(t, "concurrent mixed success/failure", 5*time.Second, func() {
		done := make(chan struct{})
		for i := 0; i < 100; i++ {
			go func() {
				defer func() { done <- struct{}{} }()
				_ = inv.Reserve("widgets", 1) // 50 succeed, 50 fail — both paths hammered
			}()
		}
		for i := 0; i < 100; i++ {
			<-done
		}
		if got := inv.Available("widgets"); got != 0 {
			t.Errorf("available = %d, want 0 (exactly 50 should have succeeded)", got)
		}
	})
}
