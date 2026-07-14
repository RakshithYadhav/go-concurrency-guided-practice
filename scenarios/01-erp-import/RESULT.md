# RESULT — OPS-1743 (fill this in as you go)

## Baseline (measure BEFORE changing anything)

```
wall time:     (paste from measure)
throughput:    (paste)
ERP rejected:  (paste)
```

## What changed

(2-4 sentences: the shape of the fix and WHY this shape — including
why the concurrency bound is exactly what it is.)

## After

```
wall time:     (paste from measure, post-fix)
throughput:    (paste)
ERP rejected:  (paste — must be 0)
speedup:       (baseline ÷ after)
```

Race gate: `.\test-race.ps1 ./scenarios/...` → (paste PASS)

## Four-part package → ../../../WAR-STORIES.md

Draft it here with Claude AFTER the numbers above are real, then copy
the final version into WAR-STORIES.md:

1. **Resume bullet** (formula: 3 keywords + how used + business
   reason + quantified impact):
2. **Business impact (non-technical):**
3. **Technical story:**
4. **Interview narration (60-90s):**
