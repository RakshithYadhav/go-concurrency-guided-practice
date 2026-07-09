package exercises

import (
	"context"
	"testing"
	"time"
)

func TestThumbnailAll_Completes(t *testing.T) {
	images := []string{"a.png", "b.png", "c.png"}
	got, err := ThumbnailAll(context.Background(), images, func(s string) string { return "thumb-" + s })
	if err != nil {
		t.Fatalf("want nil error, got %v", err)
	}
	want := []string{"thumb-a.png", "thumb-b.png", "thumb-c.png"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q — order must match input", i, got[i], want[i])
		}
	}
}

func TestThumbnailAll_StopsOnCancel(t *testing.T) {
	// 100 images x 20ms = 2 full seconds if it ignores ctx.
	// The deadline fires at 50ms; a listening loop returns almost at once.
	images := make([]string, 100)
	for i := range images {
		images[i] = "img.png"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	got, err := ThumbnailAll(ctx, images, func(s string) string {
		time.Sleep(20 * time.Millisecond)
		return "thumb-" + s
	})
	took := time.Since(start)

	if err == nil {
		t.Fatalf("canceled at 50ms: want ctx error, got nil (rendered %d of %d)", len(got), len(images))
	}
	if took > 250*time.Millisecond {
		t.Fatalf("took %v after a 50ms deadline — the loop never checks ctx", took)
	}
	if len(got) >= len(images) {
		t.Fatalf("rendered the whole batch (%d) despite cancellation", len(got))
	}
}
