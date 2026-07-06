# Module 1 — Exercises

Your job: make this pass, with zero race reports:

```bash
go test -race ./01-goroutines/...
```

Run a single exercise while working:

```bash
go test -race ./01-goroutines/exercises -run TestCheckAll -v
```

Because concurrency bugs are probabilistic, also run the suite a few times
before calling it done:

```bash
go test -race -count=5 ./01-goroutines/exercises
```

## The exercises

| File | Task |
|------|------|
| `ex1_checkall.go` | **Implement** `CheckAll`: run health checks concurrently, results in input order. The test proves it's actually concurrent (timing) and complete. |
| `ex2_race.go` | **Fix the bug**: `WordFrequency` has a data race (shared map). Any correct fix is accepted — think about which you'd defend in review. |
| `ex3_waitgroup.go` | **Fix the bug**: `FetchAll` misuses `WaitGroup`. Understand the exact failure sequence before fixing — you'll be asked to narrate it. |
| `ex4_leak.go` | **Fix the bug**: `FirstResult` leaks goroutines. The fix is one line; knowing *why* it works is the point. (Previews channels/Module 2.) |
| `ex5_totalsize.go` | **Implement** `TotalSize`: concurrent lookups combined into ONE number — without the shared-accumulator race, and without mutexes/atomics/channels. |
| `ex6_closure.go` | **Fix the bug**: `ProcessAll`'s closures capture variables declared outside the loop. Go 1.22 does not save you here — explain why. |
| `ex7_sleep.go` | **Fix the bug**: `NotifyAll` "waits" with `time.Sleep`. The test is rigged so no sleep duration can pass — only real synchronization. |
| `ex8_append.go` | **Fix the bug**: `ValidateAll` does a concurrent, unsynchronized `append` to a shared slice — same disease as ex2's map race, different organ. |
| `ex9_earlyreturn.go` | **Fix the bug**: `BuildReports` has an early `return` that skips `wg.Done()` on some inputs — a deadlock with no panic involved. |

## Rules of engagement

- Don't change the test files, and don't change exercise function signatures.
- No `time.Sleep` as a synchronization mechanism. Ever.
- Stuck ≠ failure. Ask for a hint — you'll get a nudge, not a solution.
- When everything passes, say so and we do the PR-style review + quiz from
  `../NOTES.md`.

## Mistakes log

Every real bug made while solving these (not hypothetical ones — the actual
wrong code written, and why it broke) gets logged in
[`../MISTAKES.md`](../MISTAKES.md) as we go. It's worth rereading before
starting a new module — the same handful of patterns (loop-position vs
task-count, where a `defer` actually fires, merging shared state too early or
too late) show up again and again in new disguises.

## Original bug/skeleton, kept above every fix

Once an exercise is solved, the original buggy (or unimplemented) code isn't
deleted — it's kept as an `// ORIGINAL (before fix)` comment block directly
above the working solution, in the same file. Two reasons: it lets you (or
anyone else) re-attempt the exercise from its actual starting point later
without needing git history, and a fixed file with no visible bug teaches
nothing to someone reading it cold. When you solve a new exercise, this gets
added as part of wrapping it up — you don't need to do it yourself.

## Want more exercises?

Just ask. More can be generated anytime — either fresh scenarios on concepts
already covered (to build reps until something feels automatic) or targeted
directly at whatever's in the mistakes log. This isn't a fixed set; treat it
as bottomless until Module 1 feels solid.
