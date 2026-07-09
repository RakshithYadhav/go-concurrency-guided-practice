# Module 3 — Quiz Tracker

Oral quiz over the questions in `NOTES.md` Section 8. Graded honestly;
the module's roadmap box doesn't get checked until every question is cleared.

## Round 1 — started 2026-07-09

**Q1 (count++ three steps / lost-update timeline)** — ✅ **Pass.**
Named all three steps (LOAD, INCREMENT, STORE) after one nudge on the
missing third step. Built a correct numbered timeline with real numbers
(count 5 → 6 instead of 7 from two lost-update interleavings). Correctly
explained the mutex closes the gap by blocking a second goroutine's LOAD
until the first goroutine's full LOAD/INCREMENT/STORE + Unlock completes.

**Q2 (defer mu.Unlock() — why it matters, blast radius vs ex9)** — ✅ **Pass.**
Took several passes to get the full picture. Landed on: `defer` runs
regardless of return path; an early return without it leaves the lock held
forever; critically, EVERY future caller of `Reserve` (any item, any
goroutine) piles up behind the same `mu` and deadlocks too — not just the
original goroutine. Correctly contrasted with ex9's skipped `wg.Done()`,
which stays contained to just that one `WaitGroup` value and its `Wait()`
callers. Mutex bug is worse because the blast radius is much bigger.

**Q3 (RWMutex vs Mutex, write-while-holding-read)** — ✅ **Pass.**
Took several passes. Landed correctly on: RWMutex wins when reads vastly
outnumber writes (many readers overlap for free); it's slower in write-heavy
or uncontended workloads because it tracks a reader count + "is a writer
waiting" on every single call, unlike Mutex's single flag — cost paid whether
or not you ever cash in the overlap benefit (added this bookkeeping
explanation, with the librarian/head-count analogy, to NOTES.md Section 2).
Correctly worked out the self-deadlock: calling `Lock()` while already
holding `RLock()` waits for the reader count to hit zero, but this goroutine
IS one of those readers and is now blocked before it can reach `RUnlock()`
to decrement it — waiting on itself, forever.

## Round 2 — started 2026-07-09

**Q4 (atomics vs mutex — what each can/can't do)** — ✅ **Pass.**
Correctly narrowed it to the real distinction: atomics guarantee one
indivisible LOAD/MODIFY/STORE on a single variable, nothing more. A mutex
protects an entire critical section — arbitrary logic across as many
variables as needed. Correctly cited ex2's PriceCache (map + no separate
counter) as a case atomics alone could never handle, since check-fetch-store
needed one unbroken unit across state atomics can't reach.

**Q5 (sync.Once's two promises, why naive `if !done` breaks each)** — ✅ **Pass.**
Named both promises correctly: (1) f runs exactly once, (2) every caller
waits for f to fully finish, not just start. Explained promise-1 breakage
correctly on first pass: two goroutines can both read `done == false` before
either writes `true`, so both call f() — "imagine 100 goroutines doing this."
Promise-2 breakage took several passes — initially conflated it with the
promise-1 race. Got there by tracing the naive code's literal line order:
`done = true` runs BEFORE `f()` is called, not after. So while goroutine A
is still inside `f()` (not yet finished), `done` is already `true`. Goroutine
B checks `if !done`, sees `true`, so `!done` is `false` — B skips the block
entirely and falls straight through to `return config`, getting back a
half-built/empty result because A's `f()` hasn't written the real value yet.
Also raised (correctly, as a related but separate point set aside for this
question) that without synchronization there's no happens-before guarantee
B even sees A's write to `done` at all — a real memory-model subtlety, good
instinct, ties into Q7.

## Round 3 — started 2026-07-09

**Q6 (spin-loop, two layers)** — 🔶 **Open, deferred.** Said "not sure," then
"let's skip, this is niche." Explanation given in chat (compiler caching
`done` in a register + per-core CPU cache invisibility) but not yet answered
back in his own words. Revisit before closing out the module — this is the
theory underneath every fix made in Q1-Q5.

**Q7 (memory model + happens-before)** — 🔶 **Open, deferred.**
Got happens-before itself right (correctly reused the "handoff" analogy:
"I'm done with my work, here you go, you can use it now"). Gave only half
the memory-model rule (the "no race = no guarantee" half; missing the "no
race = behaves like one sequential order" half). Struggled with the third
part — WHY the WaitGroup Done/Wait edge specifically covers a worker's
slot-writes, not just the Done/Wait signal itself. NOTES.md Section 6 was
rewritten twice in chat (analogy version, then a concrete 2-worker/
results[]-slice numeric walkthrough) — revisit with the numeric walkthrough
next session, it wasn't fully absorbed yet. Deferred at his request.

**Q8 (channels-vs-mutexes rule + real example)** — 🔶 **Nearly closed.**
Got the rule right after one refinement, in his own words: "For a channel
it keeps moving. But for a mutex it stays put, and goroutines come and
touch it." Asked a strong depth question — "WHY would we want to move data
vs let goroutines touch shared data, WHEN does each make sense" — which
led to a new NOTES.md Section 7 subsection (assembly line vs filing
cabinet, choosing by problem shape). Still owed to close: name ONE real
channel from his Module 2 code and ONE real mutex from Module 3 code and
say why each fits its shape.

## To close out this quiz (next session)

1. **Q6:** name the two layers that can keep `for !done {}` spinning
   forever (compiler register caching + per-core CPU cache). Explanation
   given but never said back.
2. **Q7:** full memory-model rule (BOTH halves), and trace the 2-worker
   `results[]` example: why does Done/Wait guarantee main sees 10 and 20?
   Use the numeric walkthrough in NOTES.md Section 6.
3. **Q8:** the two concrete examples from his own code.

---

*Answers are graded live in chat; this file is the persistent summary so the
quiz can resume across sessions without re-asking cleared questions.*
