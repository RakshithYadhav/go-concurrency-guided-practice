package exercises

import "sync"

// Exercise 7 — FIX THE BUG: time.Sleep is not synchronization.
//
// This sends a notification to every device and collects the delivery
// receipts. The author "waited for the goroutines" with a sleep that seemed
// plenty long on their laptop. You know the rule this violates.
//
// The test is built so that NO sleep duration can ever pass it:
//   - one scenario has slow sends    -> a short sleep returns too early
//     (missing receipts, plus a race: the caller reads `receipts` while
//     goroutines are still writing it)
//   - one scenario has fast sends    -> a long sleep wastes time and fails
//     the elapsed-time assertion
// Only actual synchronization — "return exactly when the last send finishes,
// however long that takes" — satisfies both. That's the whole lesson: a sleep
// is a GUESS about how long work takes; synchronization is KNOWLEDGE that it
// finished.
//
// Your tasks:
//  1. Run the test natively: see the missing receipts. Run it under -race:
//     see the caller's read racing the workers' writes.
//  2. Fix it. You've done this twice already — this one should be quick.
//
// Do not change the function signature.

// ORIGINAL (before fix) — kept for revision / re-attempting from scratch:
//
//	func NotifyAll(devices []string, msg string, send func(device, msg string) string) []string {
//		receipts := make([]string, len(devices))
//
//		for i, d := range devices {
//			go func() {
//				receipts[i] = send(d, msg)
//			}()
//		}
//
//		time.Sleep(50 * time.Millisecond) // "sends never take longer than this... right?"
//
//		return receipts
//	}

// NotifyAll sends msg to every device concurrently; receipts[i] corresponds
// to devices[i]. BUG: it guesses how long the sends take instead of knowing.
func NotifyAll(devices []string, msg string, send func(device, msg string) string) []string {
	receipts := make([]string, len(devices))
	var wg sync.WaitGroup

	for i, d := range devices {
		wg.Add(1)
		go func() {
			defer wg.Done()
			receipts[i] = send(d, msg)
		}()
	}

	wg.Wait() // "sends never take longer than this... right?"

	return receipts
}
