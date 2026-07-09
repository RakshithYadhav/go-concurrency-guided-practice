# Mistakes Log — Module 5: context & Graceful Shutdown

Every real bug made while solving this module's exercises — what was
written, why it broke, what fixed it. Same format as Modules 1-4.
(Patterns worth rereading first: Module 4's leaks — every send needs a
second exit; length-vs-capacity; slice range needs `_, v`. New traps
native to this module: taking ctx without listening to it, forgetting
`defer cancel()`, and deriving the drain context from an already-canceled
parent.)

---

*(empty so far — entries get added as bugs actually happen)*
