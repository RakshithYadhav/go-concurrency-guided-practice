package exercises

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

var (
	errFirst  = errors.New("first failure")
	errSecond = errors.New("second failure")
	errBoom   = errors.New("boom")
)

func TestGroupLite_AllSucceed(t *testing.T) {
	g, _ := WithContext(context.Background())
	var ran atomic.Int32

	for i := 0; i < 5; i++ {
		g.Go(func() error {
			ran.Add(1)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		t.Fatalf("all succeeded: want nil, got %v", err)
	}
	if ran.Load() != 5 {
		t.Fatalf("want 5 functions run, got %d", ran.Load())
	}
}

func TestGroupLite_FirstErrorWins(t *testing.T) {
	g, _ := WithContext(context.Background())

	g.Go(func() error {
		time.Sleep(20 * time.Millisecond)
		return errFirst
	})
	g.Go(func() error {
		time.Sleep(150 * time.Millisecond)
		return errSecond
	})

	start := time.Now()
	err := g.Wait()
	took := time.Since(start)

	if !errors.Is(err, errFirst) {
		t.Fatalf("want the FIRST error (%v), got %v", errFirst, err)
	}
	if took < 100*time.Millisecond {
		t.Fatalf("Wait returned in %v — it must wait for ALL functions, not just the failed one", took)
	}
}

func TestGroupLite_CancelsRemaining(t *testing.T) {
	g, gctx := WithContext(context.Background())
	var sawCancel atomic.Int32

	for i := 0; i < 4; i++ {
		g.Go(func() error {
			select {
			case <-gctx.Done():
				sawCancel.Add(1)
				return gctx.Err()
			case <-time.After(2 * time.Second):
				return nil
			}
		})
	}
	g.Go(func() error {
		time.Sleep(30 * time.Millisecond)
		return errBoom
	})

	start := time.Now()
	err := g.Wait()
	took := time.Since(start)

	if !errors.Is(err, errBoom) {
		t.Fatalf("want %v (the first and only real error), got %v", errBoom, err)
	}
	if took > 500*time.Millisecond {
		t.Fatalf("Wait took %v — the derived context was never canceled, siblings sat out their full 2s", took)
	}
	if sawCancel.Load() != 4 {
		t.Fatalf("want all 4 siblings to see cancellation, got %d", sawCancel.Load())
	}
}

func TestGroupLite_ZeroValueWaitBlocksUntilDone(t *testing.T) {
	var g Group // zero value, no context
	var finished atomic.Bool

	g.Go(func() error {
		time.Sleep(150 * time.Millisecond)
		finished.Store(true)
		return nil
	})

	if err := g.Wait(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	if !finished.Load() {
		t.Fatal("Wait returned before the goroutine finished")
	}
}
