package exercises

// Exercise 3 — FIX THE BUG: the producer that never gets to go home.
//
// FirstMatch scans items with a producer goroutine and returns the first
// match. It WORKS — every test of its return value passes. But when the
// items contain MORE than one match, the producer finds the second one,
// sends it into an unbuffered channel... and nobody is listening. That
// goroutine blocks on the send forever. Leaked.
//
// This is exactly Module 1 ex4's bug wearing pipeline clothes — and this
// time you fix it the pipeline way (NOTES Section 5):
//
//   1. Trace the leak first: items = ["x", "match-a", "match-b"]. Where
//      exactly is the producer goroutine stuck after FirstMatch returns?
//   2. Fix it with a done channel + select around the send, so the
//      producer always has a second exit. The leak test counts goroutines
//      before and after 100 calls — it will catch you if any stay stuck.
//
// A giant buffered channel also silences the test. That's the crutch, not
// the lesson — the done-channel fix works no matter how many items there
// are, a buffer only works until it's one slot too small.
//
// Do not change the signature.

// ORIGINAL (before fix): plain send, no second exit —
//
//	func FirstMatch(items []string, match func(string) bool) (string, bool) {
//		results := make(chan string)
//		go func() {
//			defer close(results)
//			for _, it := range items {
//				if match(it) {
//					results <- it // second match: blocks forever, nobody receives
//				}
//			}
//		}()
//		v, ok := <-results
//		return v, ok
//	}
//
// Solved correctly on the first attempt: a done channel + select gave the
// producer a second way out once the consumer (the single `<-results`)
// had already returned.

// FirstMatch returns the first item for which match returns true.
func FirstMatch(items []string, match func(string) bool) (string, bool) {
	results := make(chan string)
	done := make(chan struct{})
	defer close(done)

	go func() {
		defer close(results)
		for _, it := range items {
			if match(it) {
				select {
					case results <- it:
					case <-done:
						return
				}
			}
		}
	}()
	v, ok := <-results
	return v, ok
}
