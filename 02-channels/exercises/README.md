# Module 2 — Exercises

Definition of done, same as always:

```powershell
go test ./02-channels/...                 # fast native iteration
.\test-race.ps1 -count=5 ./02-channels/...   # the real gate
```

Several of these tests guard against hangs with their own internal timeouts
(the `select { case <-done: ... case <-time.After(...) }` pattern from
Module 1's ex9) — so a deadlock shows up as a clear failure message, not a
frozen terminal. Still, pass `-timeout 30s` to `go test` while iterating.

## The exercises

| File | Task |
|------|------|
| `ex1_generate.go` | **Implement** `Generate`: your first producer — send values down a channel, close it properly, return a receive-only view. |
| `ex2_neverclosed.go` | **Fix the bug**: `CollectSquares` never returns. Nothing panics, nothing errors — it just hangs. Diagnose from the symptom. |
| `ex3_first.go` | **Implement** `FirstResponse`: fastest replica wins, with a timeout — and WITHOUT leaking the losers (Module 1 ex4's lesson, now with `select`). |
| `ex4_merge.go` | **Implement** `Merge` (fan-in): combine N input channels into one output, closed exactly when all inputs are done. Needs the multi-sender close pattern. |
| `ex5_closepanic.go` | **Fix the bug**: `StreamAll` panics. Read the panic message, find which close rule is being violated, restructure so exactly one goroutine closes. |

Suggested order: ex1 → ex2 → ex5 → ex3 → ex4 (roughly increasing
coordination complexity).

Also do `../DRILLS.md` — predict each snippet's behavior on paper BEFORE
running anything. Prediction is the skill interviews actually test.

## Rules of engagement

Same as Module 1, plus one new hard rule:

- Don't change test files or exercise function signatures.
- No `time.Sleep` as synchronization. (In THIS module that includes "sleep a
  bit so the channel is probably ready" — every one of those has a proper
  channel-shaped answer.)
- **Producers return `<-chan T` (receive-only), and the goroutine that sends
  is the one responsible for closing** — the compiler and the tests will
  both hold you to it.
- Solved exercises keep their original bug above the fix as an
  `// ORIGINAL (before fix)` block — added automatically at wrap-up.
- Mistakes go in `../MISTAKES.md` as they happen.
- Hints on request; solutions never.
