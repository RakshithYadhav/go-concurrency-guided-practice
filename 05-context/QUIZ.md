# Module 5 — Quiz Tracker

Oral quiz over the questions in `NOTES.md` Section 7. Graded honestly;
the module's roadmap box doesn't get checked until every question is
cleared.

## Round 1 — not started

1. What three things does a context carry? Explain the tree: what does
   canceling a parent do, and why doesn't cancellation flow upward?
2. Why must you `defer cancel()` even for a WithTimeout context that
   cancels itself? What exactly leaks if you don't?
3. `context.Canceled` vs `context.DeadlineExceeded` — how does each
   happen, and why would production code treat them differently? What do
   WithCancelCause/Cause add?
4. Your function takes ctx but runs a pure-CPU loop for 30 seconds. Why
   does cancellation not affect it, and what are the three ways code
   actually LISTENS to a context?
5. errgroup vs WaitGroup: name the three things errgroup adds. When one
   goroutine in a group fails, what happens to the others — and what must
   THEY do for that to work?
6. Walk the four steps of graceful shutdown in order. Why is the drain
   context derived from Background instead of the signal context? Why
   does the drain need its own deadline at all?
7. `context.WithValue`: what belongs in it, what never does, and why is
   hiding a function parameter in a context worse than an explicit
   argument?
8. Module 4's done channel and ctx.Done() — same idea. What does context
   add that made it the universal standard, and name a real API where the
   library cancels I/O for you when ctx dies.

---

*Answers are graded live in chat; this file is the persistent summary so
the quiz can resume across sessions without re-asking cleared questions.
Also still open: Module 3 quiz Q6-Q8 (`03-sync/QUIZ.md`) and the whole
Module 4 quiz (`04-patterns/QUIZ.md`).*
