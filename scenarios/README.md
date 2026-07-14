# War-Story Scenarios — Track 1

Six production problems. Each one is a real pattern that real backend
teams hit — nothing here is invented puzzle-hardness. Each closes with
measured numbers and a resume-ready story in
[`../../WAR-STORIES.md`](../../WAR-STORIES.md).

## Why these six

A war story impresses an interviewer when:

1. **The problem class has a name they recognize.** Slow nightly ETL.
   Jobs lost on deploys. Retry storms. Concurrent edits corrupting
   data. Write amplification. Goroutine leaks. Interviewers have
   personally lived these — recognition builds instant credibility.
2. **There's a diagnosis step with evidence.** "How did you find it?"
   is the real interview question. Two scenarios here require pprof
   captures, one requires race-detector output, one requires load-test
   data. "I measured, then I knew" beats "I guessed."
3. **The business stakes are plain.** Every ticket says who was hurt:
   planners staring at stale schedules, users whose uploads vanished,
   an API provider threatening to revoke the key.
4. **The numbers are real.** Baseline measured FIRST, then the fix,
   then re-measure. The resume bullet uses what the harness actually
   printed on this machine.
5. **There's a tradeoff to defend.** Why 32 workers and not 300. Why
   batching didn't hurt latency. Why jitter matters. Senior signal
   lives in the "why," not the "what."

## How each scenario runs

1. Read `PROBLEM.md` — a ticket from a tech lead: symptoms, observed
   numbers, an SLO target, and a constraint that shapes the solution.
2. Run the measurement harness. Record the BASELINE in `RESULT.md`
   before touching anything.
3. Diagnose and fix. Hints only, as always. The fake infrastructure
   (marked DO NOT MODIFY) stays untouched — fix the service, not the
   world.
4. Re-run the harness and the tests. Tests assert the ticket's target.
5. Fill in `RESULT.md`, then write the four-part package into
   `WAR-STORIES.md`: resume bullet (the formula), business impact,
   technical story, interview narration.

Scale honesty: each repo runs a scaled-down synthetic version
(seconds, not hours; hundreds of records, not millions). `PROBLEM.md`
states the real-world scale it models. `RESULT.md` records what was
actually measured. On the resume these are project work — their
strength is that every number is real and every story is defensible.

## The scenarios

**Manufacturing trio** — one fictional production-scheduling SaaS
(orders, work orders, Gantt views for planners). Same domain as
Rakshith's real work history, so these stories deepen an existing
profile.

| # | Dir | Story | Concept | Keywords for the bullet |
|---|-----|-------|---------|-------------------------|
| 1 | `01-erp-import/` | The overnight ERP sync no longer finishes overnight; planners see stale schedules until 10am | Bounded worker pool (M4) | Worker Pools, Bounded Concurrency, Data Pipelines |
| 2 | `02-schedule-conflict/` | A planner drags a job on the Gantt while the auto-rescheduler touches the same work order — double-booked machines | Keyed per-entity locking (M6) + race evidence | Race Conditions, Per-Entity Serialization, Idempotent Processing |
| 3 | `03-telemetry-flood/` | Factory machines stream status events; row-per-event writes drown the store at 3× seasonal load | Size-or-age batching + backpressure (M6) | Batching, Write Amplification, Backpressure |

**Mini-projects** — standalone services, each portfolio-ready on its
own.

| # | Dir | Story | Concept | Keywords for the bullet |
|---|-----|-------|---------|-------------------------|
| 4 | `04-thumbnailer-drain/` | Every deploy silently eats ~40 in-flight uploads | Graceful drain: signal → stop intake → drain → exit (M5) | Graceful Shutdown, Zero-Downtime Deploys, Signal Handling |
| 5 | `05-api-quota/` | A price tracker gets 18% of its calls 429-banned, and its instant retries make provider outages worse | Token bucket + jittered backoff (M6) | Rate Limiting, Exponential Backoff with Jitter, Thundering Herd |
| 6 | `06-aggregator-leak/` | An API aggregator's memory climbs 2%/hour until the 3am OOM | Live pprof leak hunt + ctx cancellation (M4/M5/M7) | Goroutine Leaks, pprof Profiling, Memory Stability |

Suggested order: **1 → 4 → 5 → 2 → 3 → 6** — diagnosis difficulty
rises; the pprof-driven hunt comes last, after Module 7's tooling has
settled in.

Scenarios are scaffolded ONE AT A TIME: the next one gets built when
the current one closes. (Only `01-erp-import/` exists right now.)

## Definition of done, per scenario

- Harness target met, tests green natively AND under
  `.\test-race.ps1 ./scenarios/...`
- `RESULT.md` filled with real before/after numbers
- Four-part package added to `WAR-STORIES.md`
- Real bugs hit along the way logged in the scenario's own notes or
  the module MISTAKES.md they belong to
