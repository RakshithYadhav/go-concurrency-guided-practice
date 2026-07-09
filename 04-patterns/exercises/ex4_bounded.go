package exercises

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

// FetchAllBounded fetches all urls with at most `limit` concurrent
// fetches, returning results in input order.
func FetchAllBounded(urls []string, limit int, fetch func(string) string) []string {
	panic("implement me")
}
