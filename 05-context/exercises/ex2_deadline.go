package exercises

import "context"

// Exercise 2 — FIX THE BUG: it takes a ctx. It never listens.
//
// ThumbnailAll renders a thumbnail for every image in a batch. Someone
// dutifully added `ctx context.Context` as the first parameter — the
// linter is happy, the signature looks professional. But trace what the
// function DOES with ctx: nothing. A web request that triggered a
// 500-image batch gets canceled by the user after 50ms... and this keeps
// rendering all 500, burning CPU on work nobody will ever see.
//
// This is the most common context bug in real codebases — accepting the
// parameter without listening to it. The fix is NOTES Section 4's
// "pulse check": this loop has no channel operations, so a select has
// nothing to select on. What's the other way code notices cancellation
// inside a plain loop?
//
// Contract after your fix:
//   - ctx alive the whole time → (all thumbnails, in input order, nil)
//   - ctx canceled mid-batch  → stop promptly: (whatever is rendered so
//     far, ctx.Err())
//
// Do not change the signature. Do not shrink the work.

// ORIGINAL (before fix): took ctx as a parameter but never touched it —
//
//	func ThumbnailAll(ctx context.Context, images []string, render func(string) string) ([]string, error) {
//		out := make([]string, 0, len(images))
//		for _, img := range images {
//			out = append(out, render(img))
//		}
//		return out, nil
//	}
//
// Fixed with the "pulse check" from NOTES Section 4 — no channel
// operation exists in this loop for a select to hook into, so ctx.Err()
// is checked directly at the top of each iteration instead.

// ThumbnailAll renders every image, in order.
func ThumbnailAll(ctx context.Context, images []string, render func(string) string) ([]string, error) {
	out := make([]string, 0, len(images))
	for _, img := range images {
		if err := ctx.Err(); err!= nil {
			return out, err
		}
		out = append(out, render(img))
	}
	return out, nil
}
