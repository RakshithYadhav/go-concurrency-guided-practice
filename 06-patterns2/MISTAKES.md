# Mistakes Log — Module 6: Patterns II (Production Techniques)

Every real bug made while solving this module's exercises — what was
written, why it broke, what fixed it. Same format as Modules 1-5.
(Patterns worth rereading first: every blocking channel op needs a
second exit (M5 ex1's six-bug arc); `defer close()` at the top, not the
bottom; slice range needs `_, v`; length-vs-capacity. New traps native
to this module: holding the map lock across the key lock, deleting a
mutex someone is waiting on, sleeping after the final retry, and timers
that never get reset.)

---

## Exercise 1 — FetchAllPaced (rate-limited fetcher)

**Bug 1 — concurrency the limiter already made pointless.** Wrote
one `errgroup.Go` goroutine per URL, each calling `limiter.Wait(ectx)`
then `append(output, fetch(url))`. `TestFetchAllPaced_ResultsInOrder`
failed: `got[0] = "res-b", want "res-a"`.

Why: a `rate.Limiter` is a single shared gate — only one caller gets a
token at a time, no matter how many goroutines are blocked on `Wait`.
Spinning up N goroutines didn't make anything faster (the limiter still
serializes token hand-out one at a time); it just meant N goroutines
racing to `append` in whatever order the scheduler happened to wake
them, instead of URL order.

**Bug 2 — burst set to 5 while chasing the order bug.** Changed
`rate.NewLimiter(rate.Limit(perSecond), 1)` to burst `5` while
debugging. `TestFetchAllPaced_ActuallyPaces` then failed: 6 calls at
20/sec finished in ~51ms instead of ~250ms.

Why: burst is how many tokens can be spent instantly, before the
per-interval pacing kicks in. Burst 5 let the first 5 calls fire
back-to-back with no gap.

**Fix:** dropped `errgroup` entirely. `FetchAllPaced` is a plain
function with one `for` loop — no `go func()`, no goroutine spawned at
all. It just runs on whatever goroutine calls it. `limiter.Wait(ctx)`
at the top of each iteration, append on success, return early with
`ctx.Err()` on cancellation. Burst back to `1`. All 4 tests pass.

Lesson: goroutines buy you overlap when the WORK is what's slow (real
I/O, as in the Module 4 worker pool). Here the GATE was the bottleneck,
not the work — so concurrency added a race and zero speedup.

## Exercise 3 — Retry (backoff + jitter + budget)

**Bug 1 — success/fail branch inverted.** First draft:
`if err := op(); err != nil { return err }` — returned immediately on
FAILURE and fell through to retry on SUCCESS. Exactly backwards for a
retry loop. Fixed by flipping to `err == nil { return nil }`.

**Bug 2 — shadowed `err`, twice.** `var err error` declared outside the
loop so the final `return err` (after all attempts fail) could report
the last error. But inside the loop, `if err := op(); ...` (later
renamed to `if e := op(); ...`) used `:=`, which creates a NEW variable
scoped to the `if`. The outer `err` was never written to, so
`TestRetry_AllFail` got `<nil>` back instead of the op's real error —
twice, since renaming to `e` sidestepped the shadowing without fixing
the missing assignment.

Fix: `if err = op(); err == nil` — plain `=`, reusing the outer
variable instead of declaring a new one.

Lesson: `:=` inside an `if`/`for` always creates a fresh variable in
that inner scope, even if a same-named variable already exists outside.
If a value needs to survive past the block (here: past the whole loop,
for the final `return`), it must be assigned with `=` into the
already-declared outer variable, not re-declared with `:=`.

## Exercise 4 — Batch (size-or-age batcher)

No bugs. All 5 tests passed first try: size trigger, age trigger,
close-flushes-partial, empty input, no goroutine leak.
