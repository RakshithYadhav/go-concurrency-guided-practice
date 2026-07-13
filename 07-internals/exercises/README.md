# Module 7 Exercises — The Gauntlet

| # | File | Task | Mode |
|---|------|------|------|
| 1 | `ex1_errgrouplite.go` | Build errgroup from scratch (Go/Wait/WithContext, first error wins, cancels siblings) | ⏱ TIMED |
| 2 | `ex2_tokenbucket.go` | Build a token-bucket rate limiter (Allow + ctx-aware Wait) | ⏱ TIMED |
| 3 | `ex3_singleflight.go` | Build singleflight (N concurrent same-key callers, ONE execution) | ⏱ TIMED |
| 4 | `ex4_leakhunt.go` | Find a planted goroutine leak USING pprof, then fix it | untimed |

## The gauntlet rule (new for this module)

Exercises 1–3 simulate a real interview:

1. Read the exercise comment. Set a **25-minute timer**. 
2. Attempt COLD — no notes, no hints, no questions while the timer
   runs. Narrate out loud as you code (NOTES Section 5: the shape, the
   ownership, the exits).
3. Timer dies → normal hint mode resumes, zero shame. The reps are the
   point, not the first-try success.
4. Log every real bug in `../MISTAKES.md` either way.

Exercise 4 is untimed — it's a tooling exercise. Its own rule instead:
**no finding the bug by reading the code.** The profiler tells you; you
just have to read what it says.

## Definition of done

```powershell
go test ./07-internals/exercises/           # fast native iteration
.\test-race.ps1 ./07-internals/...          # the REAL gate (Docker must be up)
```

Both green, mistakes logged, `// ORIGINAL (before fix)` blocks added.
