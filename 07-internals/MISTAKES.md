# Mistakes Log — Module 7: Interview Gauntlet + Internals

Every real bug made while solving this module's exercises — what was
written, why it broke, what fixed it. Same format as Modules 1-6.

Traps native to this module, worth rereading before starting:
- errgroup-lite: the first-error slot is shared mutable state — many
  goroutines may fail at once (Module 3 says guard it, or use
  sync.Once). And Wait must wait for ALL, not return at first error.
- token bucket: Allow must NEVER sleep; lazy-refill math must cap at
  burst; Wait's sleep must be a select against ctx.Done(), never a
  bare time.Sleep (Module 6 ex3's rule).
- singleflight: who deletes the key, and when — delete too early and
  a late caller re-runs fn mid-flight; too late (or never) and the
  next Do returns stale results instead of re-executing. And the
  follower wake-up must fire on the ERROR path too.
- leak hunt: the fix is a second exit for the parked goroutine, not a
  bigger buffer. A buffer hides the leak until the buffer fills.

---

*(empty so far — entries get added as bugs actually happen)*
