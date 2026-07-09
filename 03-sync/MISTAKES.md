# Mistakes Log — Module 3: sync & the Memory Model

Every real bug made while solving this module's exercises — what was
written, why it broke, what fixed it. Same format as Modules 1-2.
(Recurring patterns worth rereading before starting: check-then-act gaps,
cleanup skipped by early returns, and `:=`/`=`/loop-bounds slips from the
earlier logs.)

---

## Exercise 1 — Counter (mutex + atomic), 2026-07-09

**Attempt 1 — `var` inside a struct:**
```go
type CounterStruct struct {
    var mu sync.RWMutex   // does not compile
    var count int64
}
```
Struct fields don't use `var` — that keyword is for variables in function
bodies. Fields are just `name Type`. Fixed after a hint pointing at the
`Inventory` struct in the notes.

**Attempt 2 — receiver written backwards:**
```go
func (*CounterStruct cs) Inc() { }   // does not compile
```
A receiver follows the same `name Type` order as any parameter:
`(cs *CounterStruct)`. The `*` belongs to the type, not before it.

**Attempt 3 — `atomic is undefined`:**
Used `atomic.Int64` without importing it. The import is `sync/atomic` —
a subpackage path, same style as `sync`.

**Attempt 4 — unguarded read in the mutex counter:**
```go
func (cs *CounterWithMutex) Value() int64 {
    return cs.count   // RACE: read without the lock
}
```
`Inc()` locked, but `Value()` read the same field with no lock. A read
concurrent with a locked write is still a data race — the exercise comment
even warned about this one. Fixed with RLock/RUnlock (chose RWMutex).

**Attempt 5 — infinite recursion in the atomic counter:**
```go
func (cs *CounterWithAtomic) Value() int64 {
    return cs.Value()   // calls ITSELF forever — stack overflow
}
```
Meant to read the atomic value but called the method itself instead.
Never touched `cs.count` at all. Fix: `cs.count.Load()` — the read
counterpart to `Add`. Took two rounds to spot ("sorry that was a bug").

**Note:** the spec asked for a plain `sync.Mutex`; the solution used
`sync.RWMutex` with `RLock` in `Value()`. Correct and race-clean, but a
deviation from the stated requirement — worth remembering that in ex1's
benchmark comparison, the RWMutex version pays extra bookkeeping.

## Exercise 2 — PriceCache TOCTOU, 2026-07-09

Diagnosis before coding was solid: drew the two-goroutine collision and
correctly reasoned the lock must cover the ENTIRE check-fetch-store (if
only the write were locked, a second goroutine could still see a miss
in the gap and fetch again). One side confusion cleared on the way: `mu`
is short for mutex, not "micron."

**Attempt 1 — unlock after `return` (unreachable):**
```go
c.mu.Lock()
if price, ok := c.prices[symbol]; ok {
    return price          // returns holding the lock
}
price := fetch(symbol)
c.prices[symbol] = price
return price
c.mu.Unlock()             // dead code — never runs
```
Nothing after an unconditional `return` executes. Every path out leaked
the lock.

**Attempt 2 — `defer` registered too late:**
```go
c.mu.Lock()
if price, ok := c.prices[symbol]; ok {
    return price          // still returns holding the lock
}
...
defer c.mu.Unlock()       // registered only on the miss path
return price
```
A `defer` only takes effect when execution reaches that line. The
cache-hit path returned before ever registering it. Rule locked in:
**`defer mu.Unlock()` goes on the line directly after `mu.Lock()`.**

## Exercise 3 — Inventory lock leak, 2026-07-09

No missteps. Diagnosed the planted bug from reading alone ("Reserve's
early return will not trigger the unlock, thereby deadlocking the
method") and applied the defer idiom correctly on the first try. Pattern
recognition from Module 1 ex9 + the ex2 defer lesson above.

## Exercise 4 — MyOnce, 2026-07-09

No bugs — passed all four tests including latecomers-wait on the first
attempt. One conceptual correction: called the solution a "mutex with a
naive check," but a check performed while HOLDING the lock for the whole
duration of `f()` is not the naive pattern — it's the correct one. The
naive version is the unguarded `if !done`. (The real `sync.Once` adds an
atomic fast path for cheap repeat calls; ours is correct, just unoptimized.)
