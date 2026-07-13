package exercises

// Exercise 4 — LEAK HUNT (untimed): find the bug WITH the profiler.
//
// This pipeline WORKS. Run the correctness test — it passes. The
// output is right, every time. And yet one of the three stages leaks
// a goroutine on every single call. On a real server this function
// would run per-request, and the process would bloat until the OOM
// killer took it down at 3 AM.
//
// The rule of this exercise: NO bug-hunting by staring at the code.
// You have a profiler now. Use the NOTES Section 3 recipe:
//
//   1. go test -run TestLeakHunt_NoGoroutineGrowth ./07-internals/exercises -v
//      The failing test PRINTS THE GOROUTINE PROFILE for you.
//   2. Find the stack group whose count is ~100 (one per call).
//   3. Top frames = where they're parked (chansend? chanrecv?).
//      Your frame below = which stage, which line.
//   4. Fix THAT stage only — the smallest change that gives the parked
//      goroutine a second exit. Module 4 taught you the fix.
//   5. Log in MISTAKES.md WHICH profile line gave it away.
//
// FindFirstBig returns the enriched value of the first item whose
// enrichment exceeds threshold, or -1 if none does.

// FindFirstBig runs items through feed -> enrichStage -> pickStage and
// returns the first enriched value over threshold (or -1).
func FindFirstBig(items []int, threshold int) int {
	done := make(chan struct{})
	defer close(done)

	raw := feed(done, items)
	big := enrichStage(done, raw)
	hits := pickStage(done, big, threshold)

	if v, ok := <-hits; ok {
		return v
	}
	return -1
}

// feed streams the input slice.
func feed(done <-chan struct{}, items []int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, v := range items {
			select {
			case out <- v:
			case <-done:
				return
			}
		}
	}()
	return out
}

// enrichStage computes the expensive enrichment for each item.
func enrichStage(done <-chan struct{}, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			out <- v * v
		}
	}()
	return out
}

// pickStage forwards the first value over threshold, then stops.
func pickStage(done <-chan struct{}, in <-chan int, threshold int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			if v > threshold {
				select {
				case out <- v:
				case <-done:
				}
				return
			}
		}
	}()
	return out
}
