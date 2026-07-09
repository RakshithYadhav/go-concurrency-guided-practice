# Module 4 — Quiz Tracker

Oral quiz over the questions in `NOTES.md` Section 6. Graded honestly;
the module's roadmap box doesn't get checked until every question is
cleared.

## Round 1 — not started

1. Write the standard pipeline-stage shape from memory. Who closes what,
   and what happens — stage by stage — when the first channel closes?
2. Fan-out and fan-in: define both, explain what property of channels
   makes fan-out safe with zero extra code, and name the price you pay
   in output ordering. Where did you already build fan-in in this repo?
3. Worker pool: what are the three moving parts, who closes the jobs
   channel, and why does closing the results channel need a WaitGroup
   plus a closer goroutine? How do you pick N for I/O-bound vs CPU-bound
   work?
4. A teammate writes `for _, x := range items { go handle(x) }` over an
   unbounded input. Explain what breaks at scale and give BOTH standard
   fixes, including why a buffered channel enforces a limit at all.
5. Show, with the two-goroutine trace, how "consumer returns after one
   value" leaks the producer. Then write the done-channel fix and explain
   why receiving on a closed channel makes it work as a broadcast.
6. Why does a pipeline need no mutex anywhere, even though many
   goroutines are involved? (Hint: Module 3's ownership framing —
   who owns a value at any given moment?)
7. A pipeline stage can fail per-item. How do errors travel through a
   channel-based pipeline, and WHERE should the skip-vs-stop decision
   live? What extra machinery does "stop everything on first error" need?
8. What is backpressure, which channel type gives it to you by default,
   and why is a giant buffer usually the wrong fix for a slow consumer?
   Bonus: what are the two different ways a worker pool can stop, and
   which one does closing the jobs channel give you?

---

*Answers are graded live in chat; this file is the persistent summary so
the quiz can resume across sessions without re-asking cleared questions.
Module 3's own quiz still has Q6-Q8 open — see `03-sync/QUIZ.md`.*
