package exercises

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestKeyedMutex_SameKeySerializes(t *testing.T) {
	km := NewKeyedMutex()
	var inside, peak atomic.Int64

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 25; i++ {
				km.Lock("same-key")
				cur := inside.Add(1)
				for {
					p := peak.Load()
					if cur <= p || peak.CompareAndSwap(p, cur) {
						break
					}
				}
				time.Sleep(time.Millisecond) // hold the key briefly
				inside.Add(-1)
				km.Unlock("same-key")
			}
		}()
	}
	wg.Wait()

	if p := peak.Load(); p > 1 {
		t.Fatalf("%d goroutines were inside the SAME key's critical section at once — mutual exclusion is broken", p)
	}
}

func TestKeyedMutex_DifferentKeysOverlap(t *testing.T) {
	km := NewKeyedMutex()
	var inside, peak atomic.Int64
	keys := []string{"k1", "k2", "k3", "k4"}

	var wg sync.WaitGroup
	for _, key := range keys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			km.Lock(k)
			cur := inside.Add(1)
			for {
				p := peak.Load()
				if cur <= p || peak.CompareAndSwap(p, cur) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			inside.Add(-1)
			km.Unlock(k)
		}(key)
	}
	wg.Wait()

	if p := peak.Load(); p < 2 {
		t.Fatalf("peak concurrency across 4 DIFFERENT keys was %d — different keys must not wait on each other", p)
	}
}

func TestKeyedMutex_SequentialReuse(t *testing.T) {
	km := NewKeyedMutex()
	for i := 0; i < 100; i++ {
		km.Lock("reused")
		km.Unlock("reused")
	}
}
