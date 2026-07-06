# Go Concurrency Mastery

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
