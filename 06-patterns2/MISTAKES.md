# Mistakes Log — Module 6: Patterns II (Production Techniques)

Every real bug made while solving this module's exercises — what was
written, why it broke, what fixed it. Same format as Modules 1-5.
(Patterns worth rereading first: every blocking channel op needs a
second exit (M5 ex1's six-bug arc); `defer close()` at the top, not the
bottom; slice range needs `_, v`; length-vs-capacity. New traps native
to this module: holding the map lock across the key lock, deleting a
mutex someone is waiting on, sleeping after the final retry, and timers
that never get reset.)

---

*(empty so far — entries get added as bugs actually happen)*
