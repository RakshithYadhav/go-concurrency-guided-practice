package exercises

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPriceCache_Correctness(t *testing.T) {
	c := NewPriceCache()
	fetch := func(symbol string) float64 { return float64(len(symbol)) }

	if got := c.GetOrFetch("GOOG", fetch); got != 4 {
		t.Fatalf("GetOrFetch(GOOG) = %v, want 4", got)
	}
	if got := c.GetOrFetch("GOOG", fetch); got != 4 {
		t.Fatalf("second GetOrFetch(GOOG) = %v, want 4 (cached)", got)
	}
}

// 100 goroutines, one symbol, slow fetch: the buggy version fetches many
// times (all see a miss before any stores) and races on the map.
func TestPriceCache_FetchOncePerSymbol(t *testing.T) {
	c := NewPriceCache()
	var calls atomic.Int32
	fetch := func(symbol string) float64 {
		calls.Add(1)
		time.Sleep(20 * time.Millisecond) // slow — widens the check-to-act gap
		return 99.5
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := c.GetOrFetch("TSLA", fetch); got != 99.5 {
				t.Errorf("got %v, want 99.5", got)
			}
		}()
	}
	wg.Wait()

	if n := calls.Load(); n != 1 {
		t.Fatalf("fetch ran %d times for one symbol, want exactly 1 (check-then-act race)", n)
	}
}

// Many goroutines, many symbols: heavy concurrent map traffic. The buggy
// version usually dies on the runtime's concurrent-map guard even without
// -race.
func TestPriceCache_ManySymbols(t *testing.T) {
	c := NewPriceCache()
	fetch := func(symbol string) float64 { return float64(len(symbol)) }

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				sym := fmt.Sprintf("SYM-%d", j)
				if got := c.GetOrFetch(sym, fetch); got != float64(len(sym)) {
					t.Errorf("GetOrFetch(%s) = %v", sym, got)
				}
			}
		}()
	}
	wg.Wait()
}
