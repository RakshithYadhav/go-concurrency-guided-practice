# Module 7 — Quiz Tracker

Oral quiz over the questions in `NOTES.md` Section 6. Graded honestly;
the module's roadmap box doesn't get checked until every question is
cleared.

## Round 1 — not started

1. Walk me through what `ch <- v` actually does, all three paths
   (waiting receiver / buffer room / neither).
2. Mechanically, why does receiving from a closed channel never block?
   And why does sending on a closed channel panic instead of blocking
   or silently dropping the value?
3. In the buffered-channel walkthrough, a receive freed a slot while a
   sender was parked in sendq. Who moves the parked sender's value,
   and how does FIFO order survive?
4. Draw G-M-P for GOMAXPROCS=2 with six goroutines. One blocks on a
   channel, one enters a file-read syscall. What happens to each, and
   how many OS threads are in play at the end?
5. What is work stealing — who steals, from whom, how much, and why
   half instead of one?
6. A tight `for {}` loop with no function calls: does it starve the
   other goroutines on its P? Answer for Go 1.13 and for today, and
   name the mechanism that changed.
7. Your production server's memory climbs 2% per hour. Walk through
   finding the cause with pprof — exact steps, exact URLs, what you
   look for in the output, and how you confirm the fix.
8. When does a CPU profile mislead you while a trace tells the truth?
   Give a concrete example.

---

*Answers are graded live in chat; this file is the persistent summary
so the quiz can resume across sessions without re-asking cleared
questions. Quiz backlog across the curriculum: Module 3 Q6-Q8, all of
Modules 4, 5, 6, and 7.*
