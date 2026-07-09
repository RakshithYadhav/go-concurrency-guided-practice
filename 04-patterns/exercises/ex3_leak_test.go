package exercises

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestFirstMatch_Found(t *testing.T) {
	items := []string{"cat", "dog", "elephant", "mouse"}
	got, ok := FirstMatch(items, func(s string) bool { return len(s) > 3 })
	if !ok || got != "elephant" {
		t.Fatalf("got (%q, %v), want (\"elephant\", true)", got, ok)
	}
}

func TestFirstMatch_NotFound(t *testing.T) {
	got, ok := FirstMatch([]string{"a", "b"}, func(s string) bool { return len(s) > 5 })
	if ok || got != "" {
		t.Fatalf("got (%q, %v), want (\"\", false)", got, ok)
	}
}

func TestFirstMatch_FirstWins(t *testing.T) {
	items := []string{"skip", "match-a", "match-b", "match-c"}
	got, ok := FirstMatch(items, func(s string) bool { return strings.HasPrefix(s, "match") })
	if !ok || got != "match-a" {
		t.Fatalf("got (%q, %v), want (\"match-a\", true)", got, ok)
	}
}

func TestFirstMatch_NoGoroutineLeak(t *testing.T) {
	items := []string{"x", "match-1", "match-2", "match-3", "match-4", "y"}
	match := func(s string) bool { return strings.HasPrefix(s, "match") }

	before := runtime.NumGoroutine()
	for i := 0; i < 100; i++ {
		FirstMatch(items, match)
	}

	// Give finished goroutines a moment to actually exit, then re-check a
	// few times before declaring a leak.
	deadline := time.Now().Add(2 * time.Second)
	for {
		if runtime.NumGoroutine() <= before+3 {
			return // stable — no leak
		}
		if time.Now().After(deadline) {
			t.Fatalf("goroutines before: %d, after 100 calls: %d — producers are stuck on their second send",
				before, runtime.NumGoroutine())
		}
		time.Sleep(20 * time.Millisecond)
	}
}
