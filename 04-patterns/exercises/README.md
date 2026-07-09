# Module 4 Exercises

Same rules as always: solve until `go test ./04-patterns/...` passes, then
the real gate `.\test-race.ps1 ./04-patterns/...`. Hints on request, one
nudge at a time. No solutions.

| # | File | Kind | What it proves |
|---|------|------|----------------|
| 1 | `ex1_pipeline.go` | implement | You can write pipeline stages that own and close their outputs |
| 2 | `ex2_pool.go` | implement | You can build the worker pool (the shippio shape) from scratch |
| 3 | `ex3_leak.go` | fix the bug | You can spot and fix a leaking producer with the done-channel pattern |
| 4 | `ex4_bounded.go` | implement | You can bound concurrency with a buffered-channel semaphore AND keep results in order |

Rules of engagement:

- ex2: exactly `workers` goroutines — no spawning one goroutine per job.
- ex3: fix it with the done-channel + select pattern from NOTES Section 5.
  (A giant buffer also "passes" — that's the crutch, not the lesson.)
- ex4: do NOT use a worker pool here — keep one goroutine per item, bounded
  by a semaphore. Results must come back in input order (slots, Module 1).
- Everything clean under `-race`. That's the definition of done.
