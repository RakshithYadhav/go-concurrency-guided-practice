# Module 7 Exercises

| # | File | Task | Priority |
|---|------|------|----------|
| 1 | `ex1_errgrouplite.go` | Build errgroup from scratch (Go/Wait/WithContext, first error wins, cancels siblings) | optional, low |
| 2 | `ex2_tokenbucket.go` | Build a token-bucket rate limiter (Allow + ctx-aware Wait) | optional, low |
| 3 | `ex3_singleflight.go` | Build singleflight (N concurrent same-key callers, ONE execution) | optional, low |
| 4 | `ex4_leakhunt.go` | Find a planted goroutine leak USING pprof, then fix it | **do this one** |

## Status (2026-07-14)

The timed "gauntlet" format for 1–3 is deprioritized — real interviews
are broad; solid understanding of concepts matters more than a
rehearsed cold-build of one specific data structure. These three stay
in the repo as optional exercises to pick up later, no timer, hints
allowed like any other exercise (NOTES Section 5 has the reasoning).

**Exercise 4 is the one to do now.** No timer either, but a real rule:
**don't find the bug by reading the code.** Run the failing test, read
the goroutine profile it prints, and let the profile's count + stack
point you at the exact line (NOTES Section 3 has the recipe).

## Definition of done

```powershell
go test ./07-internals/exercises/           # fast native iteration
.\test-race.ps1 ./07-internals/...          # the REAL gate (Docker must be up)
```

Both green, mistakes logged, `// ORIGINAL (before fix)` blocks added.
