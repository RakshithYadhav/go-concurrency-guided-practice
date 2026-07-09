package exercises

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// testFetch simulates a fetch that respects its context (like a real
// HTTP client would): "bad-" URLs fail after 30ms, everything else
// takes 800ms unless canceled first.
func testFetch(ctx context.Context, url string) (string, error) {
	if strings.HasPrefix(url, "bad-") {
		time.Sleep(30 * time.Millisecond)
		return "", errors.New("fetch failed: " + url)
	}
	select {
	case <-time.After(800 * time.Millisecond):
		return "ok-" + url, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func TestFetchAllOrFail_AllSucceed(t *testing.T) {
	urls := []string{"u1", "u2", "u3", "u4"}
	fast := func(ctx context.Context, url string) (string, error) { return "ok-" + url, nil }

	got, err := FetchAllOrFail(context.Background(), urls, fast)
	if err != nil {
		t.Fatalf("all succeed: want nil error, got %v", err)
	}
	for i, u := range urls {
		if got[i] != "ok-"+u {
			t.Fatalf("got[%d] = %q, want %q — results must be in input order", i, got[i], "ok-"+u)
		}
	}
}

func TestFetchAllOrFail_Empty(t *testing.T) {
	got, err := FetchAllOrFail(context.Background(), nil,
		func(ctx context.Context, url string) (string, error) { return url, nil })
	if err != nil || len(got) != 0 {
		t.Fatalf("want ([], nil), got (%v, %v)", got, err)
	}
}

func TestFetchAllOrFail_FirstErrorCancelsTheRest(t *testing.T) {
	urls := []string{"slow-1", "slow-2", "bad-3", "slow-4", "slow-5"}

	start := time.Now()
	_, err := FetchAllOrFail(context.Background(), urls, testFetch)
	took := time.Since(start)

	if err == nil {
		t.Fatal("one URL fails: want an error, got nil")
	}
	if !strings.Contains(err.Error(), "bad-3") {
		t.Fatalf("want the fetch failure as the returned error, got: %v", err)
	}
	// The failure lands at ~30ms. If the slow fetches were canceled, we
	// finish quickly; if they ran to completion, this takes the full 800ms.
	if took > 500*time.Millisecond {
		t.Fatalf("took %v — the other fetches ran to completion; the group ctx never reached them", took)
	}
}
