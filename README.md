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
