package exercises

import (
	"sync"
	"testing"
)

const (
	counterGoroutines = 50
	counterIncrements = 2_000
)

func hammer(c Counter) {
	var wg sync.WaitGroup
	for i := 0; i < counterGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < counterIncrements; j++ {
				c.Inc()
			}
		}()
	}
	wg.Wait()
}

func TestMutexCounter(t *testing.T) {
	c := NewMutexCounter()
	hammer(c)
	if got, want := c.Value(), int64(counterGoroutines*counterIncrements); got != want {
		t.Fatalf("MutexCounter = %d, want %d (lost updates)", got, want)
	}
}

func TestAtomicCounter(t *testing.T) {
	c := NewAtomicCounter()
	hammer(c)
	if got, want := c.Value(), int64(counterGoroutines*counterIncrements); got != want {
		t.Fatalf("AtomicCounter = %d, want %d (lost updates)", got, want)
	}
}

// Value must be safe to call mid-flight — under -race, an unguarded read in
// the mutex version gets reported here.
func TestCounter_ReadDuringWrites(t *testing.T) {
	for name, c := range map[string]Counter{
		"mutex":  NewMutexCounter(),
		"atomic": NewAtomicCounter(),
	} {
		done := make(chan struct{})
		go func() {
			for i := 0; i < 10_000; i++ {
				c.Inc()
			}
			close(done)
		}()
		for {
			select {
			case <-done:
				if v := c.Value(); v != 10_000 {
					t.Fatalf("%s: final Value = %d, want 10000", name, v)
				}
				goto next
			default:
				_ = c.Value() // concurrent read while increments are in flight
			}
		}
	next:
	}
}

func BenchmarkMutexCounter(b *testing.B) {
	c := NewMutexCounter()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

func BenchmarkAtomicCounter(b *testing.B) {
	c := NewAtomicCounter()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}
