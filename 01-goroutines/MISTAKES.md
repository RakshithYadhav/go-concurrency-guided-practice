# Mistakes Log — Module 1: Goroutines

Every real bug I made while solving these exercises, logged as I go. Not a
retelling of the concept (that's in `NOTES.md`) — just: what I wrote, why it
broke, what fixed it. The point is to see my own recurring patterns.

---

## Exercise 1 — `CheckAll`

**1. `results = make(...)` without declaring `results` first.**
Used `=` instead of `:=` on a brand-new variable. Compiler caught it instantly.

**2. `make([]Result, 0, len(targets))` then writing `results[i]`.**
Confused **length** with **capacity**. Length controls valid indices;
capacity is just reserved backing space. A slice of length 0 has no valid
indices yet, no matter how much capacity it has — `results[i]` was writing
out of bounds. Fix: `make([]Result, len(targets))` — length equal to the
final size, so every index `0..len-1` is valid immediately.

**3. `wg.Add(i)` instead of `wg.Add(1)` (or `Add(len(targets))` once).**
Used the loop index as the argument to `Add`, so the counter tracked "sum of
indices seen so far" instead of "number of in-flight goroutines." `Add` means
*"this many more tasks are starting"* — it has nothing to do with position in
a loop.

**4. `defer wg.Done()` placed in the loop body, not inside the goroutine.**
`defer` runs when its *enclosing function* returns. I put the defer in
`CheckAll`'s own body, so all the `Done()` calls were scheduled to fire only
when `CheckAll` itself returned — but `CheckAll` couldn't return until
`wg.Wait()` unblocked, and `Wait()` couldn't unblock until `Done()` fired.
**Deadlock**, confirmed by the goroutine dump: `CheckAll` was frozen inside
`wg.Wait()` at the exact line number, with zero worker goroutines still
running. Fix: put `defer wg.Done()` as the first line *inside* the
`go func(){...}` literal.

**5. `isBoom := go func(){...}()`** — tried to assign the result of a `go`
statement to a variable. `go` is a statement, not an expression; it never
produces a value. Straight compiler error.

**6. Missing `return results` at the end of the function.**

**Pattern to watch for:** conflating "loop position" with "task count" (#3),
and not noticing which function a `defer` actually belongs to (#4). Both are
about *where in the code something takes effect*, not *what the code says*.

---

## Exercise 2 — `WordFrequency`

This one took several real attempts, each fixing one problem while leaving
(or introducing) another. Worth keeping all of them — the sequence itself is
the lesson.

**Attempt 1 (the original bug): all goroutines write directly into one shared
`freq` map.**
Classic data race — confirmed by `-race` (read/write map access from two
goroutines) and by Go's own runtime guard (`fatal error: concurrent map read
and map write`, a crash that's specific to maps, unlike a shared `int` which
just silently computes the wrong number).

**Attempt 2: each goroutine got its own local `frequency` map — but nothing
outside the goroutine ever kept a reference to it.**
Race was gone, but so was the data: `frequency` was a local variable inside
the closure with no home once the goroutine returned, so it became
unreachable and its counts were simply lost. Result: 0 words counted.
**Lesson:** removing a race by isolating state only helps if you also plan
*how the isolated state gets back out*.

**Attempt 3: added a merge loop, but placed it immediately after `go func(){}()`,
inside the same loop iteration — before the goroutine had necessarily run at
all.** `go f()` schedules `f` for later; it does not pause for it. The merge
loop raced ahead and read `frequency` while it was still empty (or mid-write
by the goroutine) — both a **timing bug** (merging before the work existed)
and a **new data race** (reading `frequency` from the main goroutine while
the worker goroutine concurrently wrote to it).

**Attempt 4: added `wg.Wait()` — but placed it inside the per-doc loop, right
after starting that one goroutine, before moving to the next doc.**
This "fixed" the race in the most misleading way possible: it made the code
correct AND fully serial. `Add(1) → start goroutine → Wait()` inside a loop
means only one goroutine is ever alive at a time — the concurrency the
exercise asked for was gone, replaced by goroutines that provide zero
benefit (just overhead) because each one is fully waited-on before the next
starts. **Lesson:** a "fix" that passes the correctness test isn't
automatically a good fix — check whether it preserves the actual point of
the exercise (here: real concurrency).

**Final, correct version:** pre-allocate a `[]map[string]int` slice sized
`len(docs)` *before* the loop (same shape as ex1's `results` slice); every
goroutine builds its own local `frequency` map and deposits it into its own
slot `frequencies[i]` (disjoint writes, no race — same reasoning as ex1); a
single `wg.Wait()` after the *entire* loop guarantees every goroutine is done
before a final, single-threaded merge loop combines all the maps into `freq`.

**Pattern to watch for:** "shard the work per goroutine, merge once
everything is provably finished" is the general-purpose fix for *any*
shared-mutable-state race — the same shape solves the map-race here, the
lost-update-`int` race in the demo, and generalizes to slices, structs,
counters, anything. The recurring trap is doing the merge **too early**
(attempt 3) or **serializing away the concurrency** while chasing correctness
(attempt 4) instead of restructuring *when* `Wait()` is called. This pattern
has a name — **sharding** — and its efficiency trade-offs (and the mutex
alternative) are written up in full in `NOTES.md`, Section 6, "The standard
fix when writes really do collide: sharding."

---

## Exercise 3 — `FetchAll`

**`wg.Add(1)` was called INSIDE the goroutine instead of before `go`.**
`go func(){...}()` only *schedules* the goroutine — the main goroutine keeps
running its own next line without pausing for it. So after the loop finished
launching all goroutines, the main goroutine could reach `wg.Wait()` while
the counter's value depended entirely on how many goroutines had happened to
get a scheduler turn and reach their own `Add(1)` line first — anywhere from
zero to all of them. `Wait()` returning as soon as the counter hit zero (even
via a partial, in-flight count) meant results could go missing, and `Add`
racing with `Wait`'s read of the counter is itself a documented misuse of
`sync.WaitGroup` (`go vet` on Go 1.26 flags this exact shape statically).
**Fix:** move `wg.Add(1)` to the loop body, before `go`, in the main
goroutine — so it happens-before the matching `Wait()`, unconditionally.

**Pattern to watch for:** `Add` and `Wait` are not "run in the order written"
— they're two independently-scheduled pieces of code, and WaitGroup's actual
contract is that every `Add` that raises the counter above zero must
happen-before the matching `Wait()`. `Add` inside the goroutine can never
guarantee that.

---

## Exercise 6 — `ProcessAll`

**Copied the range loop's own variables (`i`, `j`) into separate variables
(`idx`, `job`) declared OUTSIDE the loop, then had every goroutine's closure
capture those outer copies instead of the range variables themselves.**
`idx`/`job` were reassigned — not redeclared — every iteration, so there was
exactly one `idx` and one `job` for the whole function, and every goroutine's
closure held a live wire to that same pair. By the time any goroutine
actually ran, the loop had usually already overwritten `idx`/`job` several
times over. Result: duplicate answers landing in the wrong slots, other slots
left at their zero value, plus `-race` flagging the reassignments racing
against the goroutines' reads.

**Why Go 1.22 didn't save this even on 1.26:** the 1.22 fix only makes the
range loop's *own* `i`/`j` fresh per iteration. It has no idea `idx`/`job`
exist — they're separate, manually-copied variables outside the loop that
keep getting overwritten. Copying a loop variable into an outer variable and
capturing *that* recreates the pre-1.22 bug by hand, on any Go version.

**Fix:** delete the outer `idx`/`job` copies; close over `i`/`j` directly.
Each iteration then gets its own private `i`/`j` under 1.22+ semantics.

**Pattern to watch for:** the 1.22 fix protects the loop's *own* iteration
variables — not anything you copy them into. If a closure captures a
variable declared outside the loop, check whether that variable is shared
across iterations before trusting it's safe, regardless of Go version.

---

## Exercise 8 — `ValidateAll`

**Same length-vs-capacity mistake as Exercise 1, twice in a row.** First pass
used `var errs []error` (nil, length 0); second pass used
`make([]error, 0, len(records))` — length 0 again, just now with reserved
capacity. Both panicked identically: `index out of range [0] with length 0`
on `errs[i] = err`. Capacity is reserved backing space for future `append`
growth; it does **not** make an index valid to assign into directly — only
length does that. Fix: `make([]error, len(records))` — length equal to the
final size, same correction as Exercise 1.

**The other half of this exercise, past the panic:** once each goroutine
writes into its own disjoint index (race-free, same reasoning as every
indexed-slot exercise so far), the slots for records that pass validation
stay at the zero value (`nil`, for an `error`). But the function's contract
is "return every error encountered" — not one slot per record. Fix: a single
threaded pass **after** `wg.Wait()` that filters the indexed slice down to
just the non-nil entries. This is the same shape as `WordFrequency`'s
post-`Wait()` merge step — "shard into slots while concurrent, reduce/filter
once safe" — just with a filter instead of a combine.

**Pattern to watch for:** length-vs-capacity confusion is apparently a
recurring one — worth deliberately double-checking any `make([]T, ...)` call
before moving on: is the second argument what I want the *valid, writable
length* to be, right now?

**Also worth remembering from this exercise's original bug (the concurrent
`append`):** a slice is safe for concurrent writes to *disjoint indices*
(`errs[i] = err` from many goroutines, as fixed above) but **not** safe for
concurrent `append` (the original bug) — `append` mutates a shared header,
not a fixed address. Map vs slice vs `append` safety is written up as a
comparison table in `NOTES.md`, Section 6, "Which shared containers are
actually safe for concurrent writes?"
