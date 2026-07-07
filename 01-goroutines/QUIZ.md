# Module 1 — Quiz Tracker

Oral quiz over the questions in `NOTES.md` Section 10. Graded honestly;
the module's roadmap box doesn't get checked until every question (and
every follow-up) is cleared.

## Round 1 — attempted 2026-07-07

**Q1 (main doesn't wait / time.Sleep)** — ✅ **Pass.**
Got the core: process exits when `main` returns, goroutines cease to exist,
WaitGroup is the fix. Tightenings noted at review: no `defer`s run in the
killed goroutines; the precise critique of `time.Sleep` is that it's a
*guess, not a guarantee* — too long wastes time, too short silently loses
work.

**Q2 (Add before go)** — ✅ **Pass, one follow-up still open.**
Correct failure narration: goroutines only scheduled → `Add` hasn't run →
main reaches `Wait()` → counter 0 → returns immediately. Wording fix
applied: `Wait` isn't "concurrent" with main — main reaches it sequentially;
the race is workers-reaching-`Add` vs main-reaching-`Wait`.

**Q3 (loop-variable capture)** — 🔶 **Partial.**
Second attempt correctly identified: the loop is the only *writer*, closures
hold a reference and *read* later, fixes are pass-as-argument or shadow
(`i := i`). First attempt had it backwards ("goroutines update the loop
variable" — no, they only read it).

## Open follow-ups (answer these to close Round 1)

1. **Q2 follow-up:** if 2 of 5 goroutines DID run their `Add(1)` before
   main reached `Wait()` (counter = 2, not 0) — is the program correct
   then? What exactly happens?
2. **Q3:** the literal famous output of the pre-1.22 three-iteration loop —
   and why that specific value (what had the loop already done by the time
   the goroutines read `i`)?
3. **Q3:** what did Go 1.22 actually change? One sentence.
4. **Q3:** the closure rule still true in 1.26, and the ex6 trap in your own
   words: where did `idx`/`job` live, and why didn't the 1.22 change protect
   them?

## Round 2 — not started (Q4: data races & -race, Q5: G-M-P scheduler)

## Round 3 — not started (Q6: goroutine stack size, Q7: golden rule + ex4)

---

*Module 2's quiz (`../02-channels/NOTES.md` Section 8, 9 questions) and
DRILLS.md are separate and also pending.*
