package exercises

import "sync"

// Exercise 4 — IMPLEMENT: bounded parallelism with a semaphore.
//
// FetchAllBounded fetches every URL concurrently — one goroutine per
// URL — but never more than `limit` fetches in flight at once, using a
// buffered channel as a semaphore (NOTES Section 4). NOT a worker pool:
// the point here is the other standard shape.
//
// Requirements:
//   1. Results come back in the SAME ORDER as urls — urls[3]'s result at
//      index 3. You solved this exact problem in Module 1: private slots,
//      no locks needed. Why is writing results[i] from many goroutines
//      race-free when each goroutine has its own i?
//   2. At most `limit` calls to fetch running at any moment — the test
//      measures peak concurrency, one goroutine per URL or not.
//   3. With limit > 1, fetches genuinely overlap (also measured).
//   4. Clean under -race.
//
// Assume limit >= 1. Do not change the signature.

// ORIGINAL (before fix): two bugs — no WaitGroup, so it returned before
// any goroutine finished, and append instead of indexed writes, which
// both scrambled the order and raced on the shared slice:
//
//	func FetchAllBounded(urls []string, limit int, fetch func(string) string) []string {
//		sem := make(chan struct{}, limit)
//		results := make([]string, 0, len(urls))
//		for _, url := range urls {
//			go func() {
//				sem <- struct{}{}
//				defer func() { <-sem }()
//				res := fetch(url)
//				results = append(results, res) // race + wrong order
//			}()
//		}
//		return results // returns before any goroutine finishes
//	}
//
// Fixed with a WaitGroup + wg.Wait() before returning, and indexed
// writes (results[i] = res) instead of append — Module 1's slots
// pattern, reused here.

// FetchAllBounded fetches all urls with at most `limit` concurrent
// fetches, returning results in input order.
func FetchAllBounded(urls []string, limit int, fetch func(string) string) []string {
	sem := make(chan struct{}, limit)
	results := make([]string, len(urls))
	var wg sync.WaitGroup
	for i, url := range urls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <- sem}()
			res := fetch(url)
			results[i] = res
		}()
	}

	wg.Wait()
	return results
}
