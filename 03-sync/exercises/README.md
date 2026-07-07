# Module 3 — Exercises

Definition of done:

```powershell
go test -timeout 60s ./03-sync/...            # fast native iteration
.\test-race.ps1 -count=5 ./03-sync/...        # the real gate
```

After ex1 passes, also run its benchmarks and READ the numbers — they're
the evidence behind this module's "measure before optimizing" advice:

```powershell
go test -bench=. -run=^$ ./03-sync/exercises
```

## The exercises

| File | Task |
|------|------|
| `ex1_counter.go` | **Implement** the same counter two ways — `MutexCounter` and `AtomicCounter` — behind one interface. Then benchmark them against each other. |
| `ex2_cache.go` | **Fix the bug**: `PriceCache` has a check-then-act (TOCTOU) race on a shared map — the classic cache bug. Two distinct symptoms; the test catches both. |
| `ex3_deadlock.go` | **Fix the bug**: `Inventory.Reserve` has an early return that keeps the lock. First call is fine; the service seizes up later. ex9's lesson, mutex edition. |
| `ex4_once.go` | **Implement** your own `MyOnce` from scratch (no `sync.Once` allowed). Both guarantees: runs exactly once, AND latecomers wait for completion. |

Suggested order: ex1 → ex3 → ex2 → ex4 (ex4 is the thinker).

## Rules of engagement

Same standing rules as Modules 1-2, plus:

- Don't change test files or exercise signatures.
- No `time.Sleep` as synchronization.
- `defer` your unlocks unless you have a stated reason not to.
- ex4 forbids `sync.Once` itself (and `OnceFunc`/`OnceValue`) — build it
  from Mutex/atomic primitives; that's the point.
- Solved exercises get their original kept above the fix as
  `// ORIGINAL (before fix)`; mistakes go in `../MISTAKES.md` as they
  happen; more exercises available on request — the set is bottomless.
- Hints on request; solutions never.
