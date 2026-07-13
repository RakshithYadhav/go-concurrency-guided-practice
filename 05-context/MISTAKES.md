# Mistakes Log — Module 5: context & Graceful Shutdown

Every real bug made while solving this module's exercises — what was
written, why it broke, what fixed it. Same format as Modules 1-4.
(Patterns worth rereading first: Module 4's leaks — every send needs a
second exit; length-vs-capacity; slice range needs `_, v`. New traps
native to this module: taking ctx without listening to it, forgetting
`defer cancel()`, and deriving the drain context from an already-canceled
parent.)

---

## Exercise 2 — Ignored ctx, 2026-07-13

No bugs — solved correctly on the first attempt. Recognized that this
loop has no channel operations (nothing for `select` to hook into), so
the fix was the "pulse check" from NOTES Section 4: `if err := ctx.Err();
err != nil { return out, err }` at the top of each iteration, not a
`select`. Good sign of picking the right one of the three listening
techniques instead of defaulting to `select` everywhere.

## Exercise 4 — ServeUntilCanceled, 2026-07-13

No bugs — solved correctly on the first attempt, including the
deliberate trap: derived the drain context from `context.Background()`,
not from the already-canceled `ctx` parameter. Getting this wrong is the
"fresh clock" mistake from NOTES Section 6 — a child of an already-dead
context is born dead, and the drain would abort instantly instead of
giving in-flight requests their real `drainTimeout` budget. All four
tests passed, including the one built specifically to catch this trap
(`TestServe_DrainsInFlightRequest`), clean under `-race`.

## Exercise 3 — errgroup fetcher, 2026-07-13

**Attempt 1 — append instead of indexed writes, same shape as Module
4's `ProcessAll` and `ex4_bounded.go`, third occurrence of this pattern:**
```go
output := make([]string, 0, len(urls))
...
output = append(output, res)   // completion order, not input order + race
```
`got[0] = "ok-u4"` instead of `"ok-u1"` — fixed with
`make([]string, len(urls))` and `output[i] = res`, using the loop index
each goroutine already owns.

**Attempt 2 — discarded the caller's ctx:**
```go
group, ectx := errgroup.WithContext(context.Background())
```
`FetchAllOrFail` receives a `ctx` parameter and threw it away, deriving
the group from an unrelated fresh root instead. If the caller's own ctx
were later canceled (their timeout, their shutdown), this function would
never notice. Attempted fix `ctx.Background()` doesn't compile —
`Background()` is a function on the `context` package, not a method on
a context value. Final fix: `errgroup.WithContext(ctx)` — pass the
received parameter straight through.

Solved correctly on the third pass, clean under `-race`, including the
timing test proving the other four fetches were actually canceled at
~30ms instead of running their full 800ms.

## Exercise 1 — Cancellable worker pool, 2026-07-13

The hardest one in this module — a long multi-round debugging arc, six
distinct bugs across the session, worth reading end to end since several
are subtle:

1. **Empty worker bodies** — first attempt was just `go func() {}()`
   three times with no logic at all; didn't compile without a return
   type either.
2. **Worker processes exactly one job, then quits** — no `for range jbs`
   loop around the select, so with `workers < len(jobs)` most jobs never
   got picked up. Fixed by wrapping the select in `for jb := range jbs`
   (Module 4's worker-pool shape).
3. **`fn(<-jbs)` as a select-case argument** — `results <- fn(<-jbs)`
   looks protected by the select, but Go must evaluate the argument
   expression BEFORE the select can choose a case. The `<-jbs` receive
   happens outside the select entirely, with no `ctx.Done()` escape.
   Fixed by receiving the job first (via the `for jb := range jbs`
   loop), then selecting only on the send: `select { case results <-
   fn(jb): case <-ctx.Done(): return ctx.Err() }`.
4. **Feeder never closed `jbs`** — even with zero cancellation, workers
   blocked forever on `for jb := range jbs` once every job was sent,
   because nothing ever signaled "no more are coming." Every test hung,
   including the simplest no-cancellation case.
5. **Wrong tool: pulse-check on a line that has a channel operation** —
   attempted `if ctx.Err() != nil { return err }` immediately before
   `jbs <- job`. The pulse-check pattern (right for ex2's CPU-only loop)
   only catches cancellation that already happened before the check; it
   does nothing if cancellation happens WHILE blocked on the send itself.
   Fixed by wrapping the send in `select { case jbs <- job: case
   <-ctx.Done(): return ctx.Err() }` instead — same shape as the worker.
6. **`close(jbs)` only reachable on the happy path** — even after fixing
   the select, `close(jbs)` sat after the loop, so the early `return
   ctx.Err()` from inside the select skipped it entirely. Any worker
   idle on the plain `for jb := range jbs` receive (not yet in its own
   select) blocked forever with no close ever coming. Final fix:
   `defer close(jbs)` at the very top of the feeder goroutine —
   guaranteed to run on every exit path, not just the bottom of the
   function. Same "defer right after the thing it cleans up" idiom from
   Module 3's mutex-unlock lesson, applied to a channel close instead
   of a lock.

All four tests passed on the final attempt, including the leak counter,
clean under `-race`.
