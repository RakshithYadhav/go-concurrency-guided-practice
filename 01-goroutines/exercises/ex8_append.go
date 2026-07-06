package exercises

// Exercise 8 — FIX THE BUG: concurrent append to a shared slice.
//
// You've now fixed a shared-MAP race (ex2). This is the same disease in a
// different organ: ValidateAll runs validation for every record concurrently
// and collects the failures into one shared slice via `append`.
//
// `append` looks harmless — it's "just adding an item" — but under the hood
// it reads the slice's current (pointer, length, capacity), computes where
// the new element goes, writes it, then writes back a new (pointer, length,
// capacity). That's a read-modify-write, exactly like `count++`. Two
// goroutines appending to the SAME slice variable at the same time can
// clobber each other's update — lost errors, or worse.
//
// Your tasks:
//  1. Run it natively a few times, then under -race. Read what's reported.
//  2. Fix it, concurrently (one goroutine per record). You already know a
//     race-free shape for "N goroutines each contribute a result" — you
//     used it to fix ex1 and ex2. It applies again here; the payload is just
//     different (zero-or-one error per record instead of a whole map).
//
// Do not change the function signature.

import "sync"

// ORIGINAL (before fix) — kept for revision / re-attempting from scratch:
//
//	func ValidateAll(records []string, validate func(record string) error) []error {
//		var errs []error
//
//		var wg sync.WaitGroup
//		for _, r := range records {
//			wg.Add(1)
//			go func() {
//				defer wg.Done()
//				if err := validate(r); err != nil {
//					errs = append(errs, err) // BUG: unsynchronized shared append
//				}
//			}()
//		}
//		wg.Wait()
//
//		return errs
//	}

// ValidateAll runs validate on every record concurrently and returns every
// error encountered (order does not matter). BUG: concurrent unsynchronized
// append to a shared slice.
func ValidateAll(records []string, validate func(record string) error) []error {
	errs := make([]error, len(records))

	var wg sync.WaitGroup
	for i, r := range records {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := validate(r); err != nil {
				errs[i] = err // BUG: unsynchronized shared append
			}else {
				errs[i] = nil
			}
		}()
	}
	wg.Wait()

	var out []error
	for _, er := range errs{
		if er != nil {
			out = append(out, er)
		}
	}

	return out
}
