package exercises

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_FirstTrySucceeds(t *testing.T) {
	calls := 0
	start := time.Now()
	err := Retry(context.Background(), 5, 100*time.Millisecond, func() error {
		calls++
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("want (nil, 1 call), got (%v, %d calls)", err, calls)
	}
	if took := time.Since(start); took > 50*time.Millisecond {
		t.Fatalf("success on try 1 took %v — there should be no waiting at all", took)
	}
}

func TestRetry_SucceedsOnThird(t *testing.T) {
	calls := 0
	start := time.Now()
	err := Retry(context.Background(), 5, 40*time.Millisecond, func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	took := time.Since(start)

	if err != nil || calls != 3 {
		t.Fatalf("want (nil, 3 calls), got (%v, %d calls)", err, calls)
	}
	// waits: ~40ms then ~80ms (plus jitter up to 50%) → at least ~120ms,
	// at most ~180ms + scheduling slack
	if took < 110*time.Millisecond {
		t.Fatalf("3 attempts with base 40ms finished in %v — backoff isn't happening (or isn't growing)", took)
	}
	if took > 400*time.Millisecond {
		t.Fatalf("3 attempts took %v — waits are far larger than base*2^n + 50%% jitter", took)
	}
}

func TestRetry_AllFail(t *testing.T) {
	calls := 0
	wantErr := errors.New("permanent")
	start := time.Now()
	err := Retry(context.Background(), 3, 20*time.Millisecond, func() error {
		calls++
		return wantErr
	})
	took := time.Since(start)

	if !errors.Is(err, wantErr) {
		t.Fatalf("want the op's last error, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("want exactly 3 calls, got %d", calls)
	}
	// waits happen only BETWEEN attempts: ~20 + ~40 (+jitter). A trailing
	// wait after the final failure would push this past ~200ms.
	if took > 200*time.Millisecond {
		t.Fatalf("3 attempts took %v — did you sleep after the LAST failure?", took)
	}
}

func TestRetry_CancelDuringWaitStopsPromptly(t *testing.T) {
	calls := 0
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := Retry(ctx, 5, 2*time.Second, func() error { // first wait would be ~2s
		calls++
		return errors.New("always fails")
	})
	took := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want context.DeadlineExceeded, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("want exactly 1 call before the canceled wait, got %d", calls)
	}
	if took > 500*time.Millisecond {
		t.Fatalf("returned after %v — the backoff wait ignores ctx (plain time.Sleep?)", took)
	}
}
