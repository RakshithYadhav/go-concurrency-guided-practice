package exercises

import (
	"fmt"
	"strings"
	"testing"
)

func TestWordFrequency(t *testing.T) {
	// Many docs, heavy word overlap: maximizes contention so the race is
	// caught by -race (and often panics with "concurrent map writes" even
	// without it).
	var docs []string
	for i := 0; i < 100; i++ {
		docs = append(docs, strings.Repeat("go is fun and go is fast ", 50))
		docs = append(docs, fmt.Sprintf("doc%d has unique words ", i))
	}

	got := WordFrequency(docs)

	// Serial reference count.
	want := make(map[string]int)
	for _, doc := range docs {
		for _, w := range strings.Fields(doc) {
			want[w]++
		}
	}

	if len(got) != len(want) {
		t.Fatalf("got %d distinct words, want %d", len(got), len(want))
	}
	for w, n := range want {
		if got[w] != n {
			t.Errorf("count[%q] = %d, want %d (lost updates?)", w, got[w], n)
		}
	}
}
