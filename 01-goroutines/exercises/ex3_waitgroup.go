package exercises

// Exercise 3 — FIX THE BUG: WaitGroup misuse.
//
// FetchAll fetches N records concurrently and returns them in order. The
// author put wg.Add(1) in what they thought was a reasonable place. The test
// fails — sometimes with missing results, sometimes with a race report,
// depending on scheduling luck.
//
// Your tasks:
//  1. Before touching the code, write down (or say out loud) the EXACT
//     sequence of events that lets Wait() return before the work is done.
//     This is a top-3 Go interview question; the fix is trivial, the
//     narration is the skill.
//  2. Fix it.
//
// Fun fact: this bug is so common that `go vet` (Go 1.26) flags the Add line
// statically. The tooling knows WHERE the bug is; your job is to explain WHY
// it's a bug. After your fix, `go vet ./...` should be clean.
//
// Do not change the function signature.

import "sync"

// FetchAll fetches every id concurrently; result[i] corresponds to ids[i].
// BUG: the WaitGroup is used incorrectly.
func FetchAll(ids []int, fetch func(id int) string) []string {
	results := make([]string, len(ids))

	var wg sync.WaitGroup
	for i, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = fetch(id)
		}()
	}
	wg.Wait()

	return results
}
