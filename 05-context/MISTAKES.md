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
