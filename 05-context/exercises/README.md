# Module 5 Exercises

Same rules as always: solve until `go test ./05-context/...` passes, then
the real gate `.\test-race.ps1 ./05-context/...`. Hints on request, one
nudge at a time. No solutions.

| # | File | Kind | What it proves |
|---|------|------|----------------|
| 1 | `ex1_ctxpool.go` | implement | Your Module 4 worker pool, now cancellable end to end |
| 2 | `ex2_deadline.go` | fix the bug | You can spot code that TAKES a ctx but never LISTENS to it |
| 3 | `ex3_errgroup.go` | implement | First-error-cancels-everyone, with errgroup |
| 4 | `ex4_shutdown.go` | implement | The real graceful-shutdown sequence for an HTTP server |

Rules of engagement:

- ex1: the pool must stop PROMPTLY on cancellation — the test gives it a
  short deadline and measures how fast it lets go. And no goroutine may
  leak on the early-exit path (counted, same as Module 4 ex3).
- ex2: fix it by making the code listen to ctx (NOTES Section 4). Do not
  shrink the work or add sleeps.
- ex3: use `errgroup.WithContext` (that's the lesson). Results in input
  order — slots, again. Pass the GROUP's ctx to fetch, or the
  first-failure cancellation can never reach the other fetches.
- ex4: the drain context must give in-flight requests their full budget.
  Think hard about WHICH parent you derive it from (NOTES Section 6 —
  the "fresh clock" subtlety). The tests catch the wrong parent.
- Everything clean under `-race`. That's the definition of done.
