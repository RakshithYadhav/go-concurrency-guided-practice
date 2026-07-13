# Module 6 — Quiz Tracker

Oral quiz over the questions in `NOTES.md` Section 7. Graded honestly;
the module's roadmap box doesn't get checked until every question is
cleared.

## Round 1 — not started

1. A worker pool of 8 already bounds concurrency. Show, with numbers,
   how it can still overwhelm a downstream API — and explain what a rate
   limiter bounds that the pool doesn't.
2. Explain the token bucket: what the rate does, what the burst does,
   and why allowing a burst at all is useful. What's the difference
   between `Wait(ctx)` and `Allow()`, and when would a server prefer
   each?
3. Keyed mutex: why is the outer map-mutex released BEFORE taking the
   per-key mutex? What exact failure comes back if you hold it across
   the key lock?
4. The delete-on-unlock cleanup in a keyed mutex silently breaks mutual
   exclusion. Walk the three-goroutine interleaving that shows two
   goroutines inside the same key's critical section.
5. Backoff without jitter caused synchronized retry waves during a real
   outage. Explain the thundering herd, why exponential backoff alone
   doesn't fix it, and what jitter changes.
6. Why must every retry wait be a select against ctx.Done() rather than
   a plain time.Sleep? What's the user-visible difference?
7. Batching: why do you need BOTH a size trigger and an age trigger?
   What breaks with only size? Only age? Where does the final
   close-flush fit?
8. Your shippio importer uses a keyed mutex per shipment and batched
   commits. For each, name the failure you'd see if it were removed.

---

*Answers are graded live in chat; this file is the persistent summary so
the quiz can resume across sessions without re-asking cleared questions.
Quiz backlog across the curriculum: Module 3 Q6-Q8, all of Module 4,
all of Module 5, all of Module 6.*
