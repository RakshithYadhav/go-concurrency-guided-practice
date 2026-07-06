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
(attempt 4) instead of restructuring *when* `Wait()` is called.
