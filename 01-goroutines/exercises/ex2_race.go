package exercises

// Exercise 2 — FIX THE BUG: data race on a shared map.
//
// WordFrequency counts word occurrences across documents, processing each
// document in its own goroutine. It compiles, and on a lucky run it even
// "works". It is also completely broken.
//
// Your tasks:
//  1. Run `go test -race -run TestWordFrequency ./01-goroutines/exercises`
//     and READ the race report — both stack traces. Understand what raced.
//  2. Fix it. Keep it concurrent (one goroutine per document). Several fixes
//     are valid (mutex around the map, per-goroutine maps merged at the end,
//     ...). Pick one and be ready to defend it in review.
//
// Do not change the function signature.

import (
	"strings"
	"sync"
)

// ORIGINAL (before fix) — kept for revision / re-attempting from scratch:
//
//	func WordFrequency(docs []string) map[string]int {
//		freq := make(map[string]int)
//
//		var wg sync.WaitGroup
//		for _, doc := range docs {
//			wg.Add(1)
//			go func() {
//				defer wg.Done()
//				for _, word := range strings.Fields(doc) {
//					freq[word]++
//				}
//			}()
//		}
//		wg.Wait()
//
//		return freq
//	}

// WordFrequency returns how many times each word appears across all docs.
// BUG: multiple goroutines mutate `freq` with no synchronization.
func WordFrequency(docs []string) map[string]int {
	freq := make(map[string]int)
	frequencies := make([]map[string]int, len(docs))
	var wg sync.WaitGroup
	for i, doc := range docs {
		wg.Add(1)
		frequency := make(map[string]int)
		go func() {
			defer wg.Done()
			for _, word := range strings.Fields(doc) {
				frequency[word]++
			}

			frequencies[i] = frequency
		}()
	}
	wg.Wait()

	for _, frequency := range frequencies {
		for k, value := range frequency {
			freq[k] += value
		}
	}

	return freq
}
