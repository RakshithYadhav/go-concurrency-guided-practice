package exercises

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFlight_SameKeyRunsOnce(t *testing.T) {
	f := NewFlight()
	var calls atomic.Int32
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			v, err := f.Do("user:42", func() (string, error) {
				calls.Add(1)
				time.Sleep(100 * time.Millisecond) // the "slow database query"
				return "answer", nil
			})
			if err != nil {
				t.Errorf("want nil error, got %v", err)
			}
			if v != "answer" {
				t.Errorf("want %q, got %q", "answer", v)
			}
		}()
	}

	close(start) // 50 callers, same key, same instant
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("50 concurrent callers: want fn to run exactly ONCE, ran %d times", calls.Load())
	}
}

func TestFlight_DifferentKeysDontSerialize(t *testing.T) {
	f := NewFlight()
	var calls atomic.Int32
	var wg sync.WaitGroup

	start := time.Now()
	for _, key := range []string{"a", "b"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f.Do(key, func() (string, error) {
				calls.Add(1)
				time.Sleep(100 * time.Millisecond)
				return key, nil
			})
		}()
	}
	wg.Wait()
	took := time.Since(start)

	if calls.Load() != 2 {
		t.Fatalf("two different keys: want 2 executions, got %d", calls.Load())
	}
	if took > 180*time.Millisecond {
		t.Fatalf("two 100ms calls on DIFFERENT keys took %v — they serialized", took)
	}
}

func TestFlight_SequentialReExecutes(t *testing.T) {
	f := NewFlight()
	var calls atomic.Int32
	fn := func() (string, error) {
		calls.Add(1)
		return "v", nil
	}

	f.Do("k", fn)
	f.Do("k", fn) // the first flight landed; this must be a NEW flight

	if calls.Load() != 2 {
		t.Fatalf("sequential calls: want 2 executions (dedup is not a cache), got %d", calls.Load())
	}
}

func TestFlight_ErrorsAreSharedToo(t *testing.T) {
	f := NewFlight()
	errDB := errors.New("database on fire")
	var calls atomic.Int32
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := f.Do("k", func() (string, error) {
				calls.Add(1)
				time.Sleep(50 * time.Millisecond)
				return "", errDB
			})
			if !errors.Is(err, errDB) {
				t.Errorf("follower must get the leader's error %v, got %v", errDB, err)
			}
		}()
	}

	close(start)
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("want 1 execution even on failure, got %d", calls.Load())
	}
}
