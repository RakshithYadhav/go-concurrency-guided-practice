package exercises
import ("errors"
"time")

// Exercise 3 — IMPLEMENT: fastest replica wins, with a timeout, no leaks.
//
// This is Module 1's ex4 (FirstResult) grown up. Same hedged-request
// pattern — query every replica, return whichever answers first — but now
// with a production requirement: if NO replica answers within `timeout`,
// give up and return an error instead of waiting forever.
//
// Requirements:
//  1. Query every replica concurrently.
//  2. Return the first response received, as soon as it arrives (don't wait
//     for the stragglers).
//  3. If no response arrives within `timeout`, return ("", an error).
//  4. NO goroutine leaks in ANY scenario — including the timeout case,
//     where every replica eventually finishes and tries to deliver its
//     answer to nobody. You solved exactly this in Module 1 ex4; the same
//     one idea covers it here.
//  5. Race-free, of course.
//
// Tools you'll want: select, time.After. (NOTES Section 6 has both.)
//
// Forbidden: time.Sleep.

// FirstResponse queries all replicas concurrently and returns the first
// answer, or an error if none arrives within timeout.
func FirstResponse(replicas []string, query func(replica string) string, timeout time.Duration) (string, error) {
	results := make(chan string, len(replicas))

	for _, replica := range replicas {
		go func() {
			results <- query(replica)
		}()
	}

	select {
	case v := <-results:
		return v, nil
	case <-time.After(timeout):
		return "", errors.New("Error")
	}
}
