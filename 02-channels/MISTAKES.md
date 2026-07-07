# Mistakes Log — Module 2: Channels

Every real bug made while solving this module's exercises — what was
written, why it broke, what fixed it. Same format as Module 1's log.
(Reread `../01-goroutines/MISTAKES.md` before starting: several of those
patterns will try to come back wearing channel costumes.)

---

## Exercise 1 — `Generate`

Six attempts total. The first five each fixed one problem while the
underlying design question — "does this need more than one sender?" — kept
getting deferred. Worth keeping the whole arc; the individual fixes matter
less than the pattern of what got addressed vs. skipped each time.

**Attempt 1: `make(chan int, len(nums))`, filled synchronously inside
`Generate`, no goroutine at all.**
The exercise explicitly forbids this by name ("buffering sized to 'fit
everything' so you can send without a goroutine"). All tests still PASSED —
a buffer exactly sized to the input means every send succeeds without
blocking, so the timing test couldn't catch it. **Lesson:** a green test
suite doesn't prove a design is right; this loophole slipped past the tests
entirely and had to be caught by re-reading the exercise's own constraints.

**Attempt 2: removed the buffer (`make(chan int)`) but still sent directly
inside `Generate`, no goroutine.**
Instant hang. Root cause: `Generate` must `return ch` before the caller can
receive, but an unbuffered send blocks until a receiver exists — and
`Generate` can't reach `return` while blocked on its own send. Two things
each waiting on the other to move first (proven with a minimal standalone
repro: `ch <- 42` then `<-ch` in one goroutine deadlocks immediately,
`fatal error: all goroutines are asleep`).

**Attempt 3: one goroutine PER VALUE, `close(ch)` called right after the
loop that launches them.**
`panic: send on closed channel`. `go func(){}()` only *schedules* a
goroutine — the loop finishes launching everyone almost instantly, so
`close(ch)` ran before any spawned goroutine reached its send. Also
introduced a second, undiscussed problem: `len(nums)` independent goroutines
racing to send on one channel gives no ordering guarantee at all, which
requirement 2 (order preserved) needs.

**Attempt 4: added a `sync.WaitGroup` — but wrote `wg.Done()` where
`wg.Wait()` was meant.**
Same panic as attempt 3, because `Done()` decrements and returns immediately
— it doesn't block for anything, so it provided zero protection against
closing too early. The one-goroutine-per-value ordering problem was still
present and still unaddressed.

**Attempt 5: removed `close(ch)` entirely (regression), kept the stray
`wg.Done()`.**
`panic: sync: negative WaitGroup counter`. Now there were 6 total `Done()`
calls (5 from the goroutines' defers + 1 stray one) against only 5 `Add(1)`
calls — an unmatched decrement, mirroring Module 1 `ex3`'s Add/Wait misuse
but on the Done side instead. Separately, with `close` gone, the function
also regressed straight back into `ex2`'s exact bug (`range` over a channel
that's never closed hangs forever) — not yet caught by a test run at this
point, but present.

**Attempt 6 (final, correct): exactly one goroutine, looping through `nums`
in order, closing once the loop finishes.**
```go
func Generate(nums ...int) <-chan int {
    ch := make(chan int)
    go func() {
        for _, num := range nums {
            ch <- num
        }
        close(ch)
    }()
    return ch
}
```
No `WaitGroup` needed at all — with exactly one sender, that same goroutine
always knows for certain when it's safe to close: the instant its own loop
ends. Passes all tests, clean under `-race` ×10.

**Pattern to watch for:** each of the first five attempts fixed the most
recently-flagged symptom without stepping back to question whether "one
goroutine per value" was ever the right shape. `ex5_closepanic`'s
multi-sender-close pattern (`WaitGroup` + coordinated close) is for
genuinely multiple independent sources; `Generate` has exactly one ordered
sequence and one logical producer, which is a `WaitGroup`-free,
single-goroutine problem (see `NOTES.md`, "The generator/producer pattern"
and "One goroutine, or N?", both added after this exercise). When a fix
doesn't fully land, it's worth asking whether the *architecture* — not just
the latest bug — needs to change, the same trap as Module 1's
`WordFrequency` attempt 4 (serializing away all concurrency to "fix" a race).

---

## Exercise 2 — `CollectSquares`

The original bug (no `close` anywhere → `range` hangs forever, no error) was
diagnosed correctly right away. The fix took one wrong intermediate stop.

**Wrong attempt: `close(squares)` added, but placed in `CollectSquares`'s
own body, immediately after starting the goroutine — not inside it.**
`panic: send on closed channel`. `go func(){...}()` only *schedules* the
goroutine; it doesn't pause for it to do anything. So `close` ran almost
instantly — before the goroutine had necessarily sent even its first
value — and whenever the goroutine did run and tried to send, the channel
was already closed. Exactly the same "who's allowed to close, and what do
they need to know first" question as `ex5`/`Merge`, just showing up a
second time in a single-sender shape instead of multi-sender.

**Fix:** move `close(squares)` to the last line *inside* the goroutine,
after its own `for` loop finishes. With exactly one sender, no `WaitGroup`
is needed at all — that goroutine alone always knows for certain when it's
safe to close: the instant its own loop ends. Passes both tests, clean
under `-race` ×10.

**Pattern to watch for:** "add a `close` call" is not, by itself, a fix —
*where* it's placed relative to the actual sending is the entire question.
A `close` placed anywhere other than "the last thing the sender does, after
it's truly done sending" either closes too early (this bug) or never closes
at all (the original bug) — there's no safe middle option for a
single-sender producer.

---

## Exercise 4 — `Merge` (fan-in)

Four attempts, each surfacing a genuinely different concept — good exercise
for stress-testing everything the module covers at once.

**Attempt 1: `for input := range inputs` (single loop variable) on a slice
of channels.**
Ranging over a slice with one variable gives the **index**, not the
element — `input` was an `int` (0, 1, 2...), not a channel at all. Line
`results <- input` was sending loop indices into the output, completely
ignoring the actual input channels' contents. Compounded by a second bug in
the same attempt: `wg.Done()` called directly in `Merge`'s body (meant to be
`wg.Wait()`) right before `close(results)` — closing with no real guarantee
any goroutine had finished, `panic: send on closed channel`.

**Attempt 2: fixed both of the above (`for _, input := range inputs`,
`wg.Done()` → `wg.Wait()`) — but kept `wg.Wait()` (and therefore `close`)
directly in `Merge`'s own body, before `return results`.**
Total hang. Goroutine dump showed `Merge` itself frozen at `wg.Wait()`,
while the 3 worker goroutines it started were frozen trying to send —
because nobody could receive from `results` until `Merge` returned it, and
`Merge` couldn't return until those same goroutines finished. Identical
catch-22 shape to `Generate`'s early attempts, just with a `WaitGroup`
standing in for the direct send. **Lesson:** *any* code that must run before
`Merge`'s `return` and also depends on something only possible *after* that
`return` recreates this deadlock — doesn't matter whether it's a raw send or
a `WaitGroup`-gated one.

**Attempt 3: moved `wg.Wait(); close(results)` into its own separate
goroutine — but each worker still did a single `v := <-input; results <- v`
instead of draining the whole channel.**
Deadlock gone, `Merge` returned immediately — but now a correctness failure:
`TestMerge_SingleInput` fed in `7, 8, 9` and got back only `[7]`. Each worker
received exactly once from its input, forwarded that one value, then exited
(hitting its `defer wg.Done()`) — never looping back to drain the rest of
that channel.

**Attempt 4 (final, correct):** each worker uses a proper drain loop
(`for { v, ok := <-input; if !ok { break }; results <- v }`, equivalent to
`range input`) so it forwards *every* value from its own channel before
exiting; a separate goroutine does `wg.Wait(); close(results)`; `Merge`
itself returns `results` immediately, with nothing blocking it. Passes all
tests, clean under `-race` ×10.

**Pattern to watch for:** this exercise needed three genuinely distinct
concepts to land at once — (1) one goroutine *per independent source*, not
one *shared* loop or one *per value* (see `NOTES.md`, "One goroutine, or
N?"); (2) each of those goroutines must fully **drain** its channel with a
loop, not take a single value; (3) the `WaitGroup`-then-close coordination
must live in a goroutine that is neither a worker nor `Merge`'s own body,
so `Merge` can return before that coordination finishes. Missing any one of
the three produces a different failure (wrong data, deadlock, or partial
data) — worth checking all three explicitly next time a fan-in-shaped
function is needed, rather than fixing one and assuming the others are fine.

---

## Exercise 6 — `Fibonacci`

Four attempts, each addressing one distinct layer — pure Go syntax, then
correctness of the actual math, then two separate edge-case failures caused
by the same root design flaw.

**Attempt 1: `for i = 0, i < n, i++`** — commas instead of semicolons
between the loop's three clauses, and `=` instead of `:=` on an undeclared
`i`. Compiler: `syntax error: unexpected ++, expected {`. Go's C-style `for`
needs `init; condition; post` with semicolons — commas don't separate
clauses at all, so the parser tried to read the whole thing as one
expression and choked on `i++`. Same `:=`-vs-`=` rule as Module 1 `ex1`'s
very first mistake, recurring.

**Attempt 2: syntax fixed, but `fib <- (i - 1) + (i - 2)` — used the loop
index itself as if it were the previous two Fibonacci values.**
`(i-1)+(i-2)` simplifies to `2i-3`, a straight line — it only coincidentally
matched the real sequence at `i=2`, then diverged immediately
(`0 1 1 3 5 7 9...` instead of `0 1 1 2 3 5 8...`). The bug: nothing was
actually *remembered* between iterations — the formula derived a value from
*position*, not from what was genuinely sent last. Also present in this
attempt: `fib <- 0` and `fib <- 1` ran unconditionally before checking `n`
at all, so `Fibonacci(0)` returned `[0 1]` instead of nothing, and
`Fibonacci(1)` returned `[0 1]` instead of `[0]`.

**Attempt 3: introduced a `fibA []int` slice with a real recurrence
(`fibA[i-1] + fibA[i-2]`) — correct math this time — but `fibA[0] = 0` and
`fibA[1] = 1` were still hardcoded, unconditional lines before the loop.**
`make([]int, n)` with `n=0` gives a zero-length slice; writing `fibA[0]`
into it panicked: `index out of range [0] with length 0`. Fixing the *math*
didn't fix the underlying issue from attempt 2 — the first two values were
still special-cased outside any loop tied to `n`.

**Attempt 4: moved the base cases inside an `if i == 0 || i == 1` branch —
but the surrounding loop's own init clause still said `i := 2`.**
Since `i` could never actually be `0` or `1` inside a loop starting at `2`,
that branch was dead code — unreachable no matter what `n` was. Net effect:
`fibA[0]`/`fibA[1]` silently stayed at their zero-value default (`0`, `0`
instead of `0`, `1`), every later `else` computation inherited that wrong
foundation (`fibA[2] = 0+0 = 0`), and the loop never even ran for `n=1`
(`2 < 1` is false), so `Fibonacci(1)` returned nothing.

**Fix:** change the loop's own start to `i := 0`, so the `if i == 0 || i ==
1` branch (whose *contents* were correct since attempt 4) can actually
execute. Passes all 5 tests, clean under `-race` ×10.

**Pattern to watch for:** three of these four attempts were fixed at the
wrong layer — attempt 2 tried to patch the *math* while leaving the
*structural* problem (hardcoded values outside any `n`-driven loop) intact;
attempt 4 tried to patch the *structure* (an `if` for the base cases) while
leaving a *contradicting* loop bound in place right next to it. Worth a
habit: when adding a special-case branch inside a loop, immediately check
whether the loop's own bounds can actually reach that branch at all — code
that "looks like" it handles a case but can never execute it is easy to
miss on read-through and only shows up as a test failure.
