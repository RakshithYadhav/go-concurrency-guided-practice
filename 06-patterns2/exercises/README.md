# Module 6 Exercises

Same rules as always: solve until `go test ./06-patterns2/...` passes,
then the real gate `.\test-race.ps1 ./06-patterns2/...`. Hints on
request, one nudge at a time. No solutions.

| # | File | Kind | What it proves |
|---|------|------|----------------|
| 1 | `ex1_ratelimit.go` | implement | You can pace calls to a rate, not just bound concurrency |
| 2 | `ex2_keyedmutex.go` | fix the bug | You can spot the delete-recreate race that silently breaks per-key locking |
| 3 | `ex3_retry.go` | implement | You can write a production retry: backoff, jitter, budget, ctx-aware waits |
| 4 | `ex4_batch.go` | implement | You can batch by size AND age with a clean close-flush, leak-free |

Rules of engagement:

- ex1: `time.Ticker` or `rate.Limiter`, your choice — the test only
  measures the pacing. Results in input order (slots). Ctx-aware: a
  canceled ctx must stop the run promptly.
- ex2: the fix must keep mutual exclusion for the SAME key while still
  letting DIFFERENT keys overlap (both are tested). The simple fix is
  acceptable (see NOTES Section 3); a reference-counted cleanup is the
  stretch goal.
- ex3: every wait must be a select against ctx.Done() — a plain
  time.Sleep fails the cancellation test. Jitter must be present (the
  comment says how much); the growth of the gaps is what's tested.
- ex4: one goroutine, one select loop, a timer you reset — NOTES
  Section 5 describes every piece. Watch the close-flush and the
  "timer starts with the batch's first item" detail.
- Everything clean under `-race`. That's the definition of done.
