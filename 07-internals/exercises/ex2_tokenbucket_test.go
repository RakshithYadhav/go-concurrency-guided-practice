package exercises

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenBucket_BurstSpendsInstantly(t *testing.T) {
	tb := NewTokenBucket(10, 3)

	start := time.Now()
	for i := 1; i <= 3; i++ {
		if !tb.Allow() {
			t.Fatalf("call %d: want true (bucket starts full with burst=3), got false", i)
		}
	}
	if tb.Allow() {
		t.Fatal("call 4: want false (burst spent, no time has passed), got true")
	}
	if took := time.Since(start); took > 50*time.Millisecond {
		t.Fatalf("4 Allow calls took %v — Allow must not sleep", took)
	}
}

func TestTokenBucket_RefillsAtRateCappedAtBurst(t *testing.T) {
	tb := NewTokenBucket(20, 1) // one token per 50ms, holds at most 1

	if !tb.Allow() {
		t.Fatal("new bucket should start full")
	}
	if tb.Allow() {
		t.Fatal("bucket drained; an immediate second Allow must be false")
	}

	time.Sleep(130 * time.Millisecond) // ~2.6 tokens accrue — but burst caps at 1

	if !tb.Allow() {
		t.Fatal("130ms at 20/sec: at least one token accrued, want true")
	}
	if tb.Allow() {
		t.Fatal("burst is 1: no matter how long you wait, only ONE token can be stored")
	}
}

func TestTokenBucket_WaitPaces(t *testing.T) {
	tb := NewTokenBucket(20, 1) // 50ms per token after the first
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < 5; i++ {
		if err := tb.Wait(ctx); err != nil {
			t.Fatalf("Wait %d: %v", i, err)
		}
	}
	took := time.Since(start)

	// first is free (full bucket), then 4 gaps of ~50ms
	if took < 150*time.Millisecond {
		t.Fatalf("5 Waits at 20/sec took %v — that's not paced", took)
	}
	if took > 600*time.Millisecond {
		t.Fatalf("5 Waits took %v — way over budget, Wait is over-sleeping", took)
	}
}

func TestTokenBucket_WaitHonorsCtx(t *testing.T) {
	tb := NewTokenBucket(1, 1) // after draining, next token is a full second away
	if !tb.Allow() {
		t.Fatal("new bucket should start full")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := tb.Wait(ctx)
	took := time.Since(start)

	if err == nil {
		t.Fatal("ctx died at 80ms with the next token 1s away: want an error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want context.DeadlineExceeded, got %v", err)
	}
	if took > 300*time.Millisecond {
		t.Fatalf("Wait returned after %v — the wait outlived its context", took)
	}
}

func TestTokenBucket_AllowNeverBlocks(t *testing.T) {
	tb := NewTokenBucket(1, 1)
	tb.Allow() // drain

	start := time.Now()
	for i := 0; i < 10_000; i++ {
		tb.Allow()
	}
	if took := time.Since(start); took > 250*time.Millisecond {
		t.Fatalf("10k failed Allow calls took %v — Allow must return immediately, not wait", took)
	}
}

func TestTokenBucket_ConcurrentSafe(t *testing.T) {
	tb := NewTokenBucket(5, 5)
	var granted atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if tb.Allow() {
					granted.Add(1)
				}
			}
		}()
	}
	wg.Wait()

	// 5 burst tokens exist; the test runs in a few ms, so refill can
	// add at most a couple more. Far fewer than the 500 attempts.
	got := granted.Load()
	if got < 5 || got > 8 {
		t.Fatalf("500 concurrent attempts on a burst-5 bucket: want 5-8 grants, got %d", got)
	}
}
