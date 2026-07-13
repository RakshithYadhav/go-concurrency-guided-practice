# Go Concurrency Mastery

> **Track 1 of the Backend Mastery roadmap** — see [`../BACKEND-MASTERY.md`](../BACKEND-MASTERY.md)
> for the full 11-track plan this belongs to.

Hands-on curriculum for mastering Go concurrency the way real backend engineers
use it: worker pools, pipelines, graceful shutdown, keyed locking — plus the
interview-grade theory and runtime internals behind them.

## How each module works

1. **Explain** — read `NOTES.md`, run the programs in `demo/` and observe.
2. **Exercise** — implement the skeletons in `exercises/` until
   `go test -race ./...` passes. No peeking at solutions; ask for hints.
3. **Review** — code gets reviewed like a PR, then a short interview-style quiz.

```powershell
go run ./01-goroutines/demo/01-no-wait     # run a demo (native, no -race needed)
go test ./01-goroutines/...                # quick native test run (NO race detection)

.\test-race.ps1 ./01-goroutines/...        # the REAL gate: -race via Docker
.\test-race.ps1 -run TestCheckAll ./01-goroutines/exercises   # single test
```

> **Always finish with `-race`.** A passing test without the race detector
> proves nothing about concurrent code.
>
> On this machine `-race` runs inside a `golang:1.26` Linux container
> (Windows needs a C toolchain for the race detector; Docker sidesteps it).
> `test-race.ps1` wraps the `docker run` — Docker Desktop must be running.
> Native `go test` still works for fast iteration; the Docker run is the
> definition of done.

## Roadmap

- [ ] **Module 1 — Goroutines & the runtime**: goroutine lifecycle, `sync.WaitGroup`,
      closure capture, G-M-P scheduler, race detector
- [ ] **Module 2 — Channels**: buffered/unbuffered semantics, close, nil channels,
      the channel axioms, `select`
- [ ] **Module 3 — sync & the memory model**: Mutex/RWMutex/atomic/Once/Cond,
      happens-before, channels-vs-mutexes
- [ ] **Module 4 — Patterns I**: generators, pipelines, fan-out/fan-in, worker pools,
      goroutine leaks
- [ ] **Module 5 — context & graceful shutdown**: cancellation, timeouts, errgroup,
      signal → drain → exit
- [ ] **Module 6 — Patterns II**: semaphores, rate limiting, backpressure,
      keyed mutex, batching
- [ ] **Module 7 — Interview gauntlet + internals**: classic problems under time
      pressure, `hchan`, work-stealing, traces & pprof
- [ ] **Capstone — Webhook Delivery Service**: async queue + worker pool, per-endpoint
      ordered delivery, retry/backoff, rate limits, graceful drain

## Progress log

| Date | What happened |
|------|---------------|
| 2026-07-05 | Repo scaffolded; Module 1 opened |
| 2026-07-06 | Module 1: 7 of 9 exercises solved (ex4, ex9 + quiz pending); MISTAKES.md started |
| 2026-07-06 | Module 2 (channels) opened: notes, 4 demos, 5 exercises, 10 drills |
| 2026-07-06 | Module 2: all 6 exercises solved (ex6 Fibonacci added mid-module for extra reps), clean under `-race`; DRILLS.md + module quiz still pending. Module 1 ex4 also closed out. |
| 2026-07-07 | Module 1 quiz round 1 (Q1-Q3): Q1-Q2 passed, Q3 partial — tracker + open follow-ups in `01-goroutines/QUIZ.md`. Module 3 (sync & memory model) opened: notes, 4 demos, 4 exercises. |
| 2026-07-09 | Module 3: all 4 exercises solved, clean under `-race`. Quiz: Q1-Q5 passed, Q6-Q8 open (tracked in `03-sync/QUIZ.md`). NOTES.md grew from quiz questions: RWMutex bookkeeping cost, memory-model rewrite with notepad-handoff analogy, channels-vs-mutex "problem shape" section. Module 4 (Patterns I: pipelines & pools) opened: notes, 4 demos, 4 exercises. |
| 2026-07-09 | Module 4: all 4 exercises solved, clean under `-race` (ex3 leak fix on first attempt, others needed 1-2 hint rounds — slice-range-as-index bug hit twice, length-vs-capacity a third time, missing WaitGroup). Extensive first-principles Q&A grew both `04-patterns/NOTES.md` (fan-out isn't a race, closer-goroutine deadlock trace, error propagation, backpressure) and `02-channels/NOTES.md` (new "why blocking happens" section: open-empty-receive, closed-never-blocks, close-as-broadcast, unbuffered-send-needs-a-receiver). Module 4 quiz not yet attempted. |
| 2026-07-09 | Module 5 (context & graceful shutdown) opened: notes (context tree, defer cancel, Canceled-vs-DeadlineExceeded, WithCancelCause/WithoutCancel/AfterFunc, the 3 ways code listens to ctx, errgroup, 4-step graceful shutdown + fresh-clock trap), 4 demos (tree, timeout, errgroup-cancels-the-rest, live HTTP drain), 4 exercises (cancellable pool, fix-the-ignored-ctx, errgroup fetcher, ServeUntilCanceled). Added `golang.org/x/sync` dependency, vendored for offline Docker `-race` runs. QUIZ.md files pre-created for Modules 4 and 5. |
| 2026-07-13 | Module 6 (Patterns II: production techniques) opened: notes (concurrency-limit vs rate-limit distinction, token bucket + x/time/rate, keyed mutex + the delete-recreate trap, retries with backoff/jitter/budget + thundering herd, batching by size-or-age, or-done/tee/bridge survey), 4 demos (paced calls timestamped, same-key-vs-different-keys timed, jittered retry gaps, batch flushes labeled by reason), 4 exercises (rate-limited fetcher, FIX-the-broken-keyed-mutex, Retry with ctx-aware waits, size-or-age Batcher). Added + vendored `golang.org/x/time`. Planted keyed-mutex bug verified failing loudly (unlock panic under same-key contention). |
| 2026-07-13 | Module 5: all 4 exercises solved, clean under `-race`. ex2 and ex4 correct first try (including ex4's deliberate fresh-clock trap). ex3 took 3 passes (append/order race, discarded caller ctx, a syntax slip). ex1 was the hardest exercise in the curriculum so far — 6 distinct bugs across a long debugging arc (one-shot workers, a select-case argument evaluated outside the select's protection, an unclosed feeder channel, a pulse-check used where select was needed, close(jbs) unreachable from an early return) — solved correctly in the end via `defer close(jbs)` + select on the send. NOTES.md grew substantially from Q&A: a "why context exists" story (doomed DB queries after a closed tab), a full errgroup rewrite with a "friends checking stores" analogy, and a "three questions about this snippet" subsection on the shutdown code (cooperative cancellation / closing-bell analogy, why blocking main doesn't freeze the server, why ListenAndServe needs `go func()`). |
| 2026-07-14 | Module 6: all 4 exercises solved (native `go test` clean; `-race` gate still pending — Docker Desktop was down this session). ex1 (paced fetcher) took the longest: first draft used one `errgroup` goroutine per URL, which broke both ordering (concurrent `append` has no notion of input order) and pacing (bumped burst to 5 while chasing that, which let 5 calls through instantly) — fixed by dropping concurrency entirely, since the rate limiter already serializes everything to one token at a time; ended up a plain sequential loop, no `go` keyword anywhere. ex2 (keyed mutex) — Rakshith found and fixed the planted delete-on-unlock bug himself, unprompted. ex3 (retry) had a real inverted-condition bug (returned on failure, retried on success) plus a shadowed-`err` bug hit twice (`:=` inside the loop kept creating a new variable instead of writing into the outer one). ex4 (batcher) solved clean, first try, all 5 tests. Quiz backlog unchanged: Module 3 Q6-Q8, all of Modules 4, 5, and 6. |
