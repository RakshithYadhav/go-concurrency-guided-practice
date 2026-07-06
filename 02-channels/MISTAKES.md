# Mistakes Log — Module 2: Channels

Every real bug made while solving this module's exercises — what was
written, why it broke, what fixed it. Same format as Module 1's log.
(Reread `../01-goroutines/MISTAKES.md` before starting: several of those
patterns will try to come back wearing channel costumes.)

---

## Exercise 1 — `Generate` (in progress)

Five attempts so far, each fixing one problem while the underlying design
question — "does this need more than one sender?" — kept getting deferred.
Worth keeping the whole arc; the individual fixes matter less than the
pattern of what got addressed vs. skipped each time.

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

**Pattern to watch for:** each attempt fixed the most recently-flagged
symptom without stepping back to question whether "one goroutine per value"
was ever the right shape. `ex5_closepanic`'s multi-sender-close pattern
(`WaitGroup` + coordinated close) is for genuinely multiple independent
sources; `Generate` has exactly one ordered sequence and one logical
producer, which is a `WaitGroup`-free, single-goroutine problem (see
`NOTES.md`, "The generator/producer pattern," added after this exercise).
When a fix doesn't fully land, it's worth asking whether the *architecture*
— not just the latest bug — needs to change, the same trap as Module 1's
`WordFrequency` attempt 4 (serializing away all concurrency to "fix" a race).
