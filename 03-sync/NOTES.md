# Module 3 — The sync Package & the Memory Model

*Read top to bottom, running each demo at the ▶ Run markers. This module
assumes Modules 1-2: goroutines, WaitGroup, closures, races, channels.*

---

## 0. The problem this module solves

You already know two ways for goroutines to work with shared data safely:

1. **Don't share** — each goroutine writes its own slot / private map, merge
   after `wg.Wait()` (the sharding pattern, Module 1).
2. **Hand values over a channel** — ownership moves with the value (Module 2).

Both sidestep sharing. But sometimes the natural shape of the problem IS
"one shared thing, touched by many goroutines, alive the whole time": a
cache every request consults, a counter every handler increments, a config
that loads once. Restructuring those into sharding or channels is possible
but often awkward and slower. This module is the third tool: **keep the
shared thing, and put a lock (or an atomic operation) around access to it.**

By the end you'll have a working decision framework for *which* of the three
tools fits a given problem — that framework question is a top-tier interview
staple.

## 1. sync.Mutex — the lock

**Mutex = MUTual EXclusion.** A mutex guards a *critical section*: a stretch
of code only one goroutine may be inside at a time.

```go
var mu sync.Mutex

mu.Lock()     // blocks until the lock is free, then takes it
count++       // the critical section — at most ONE goroutine here at a time
mu.Unlock()   // releases it; one blocked waiter (if any) gets it next
```

Analogy: a single-key restroom. The key hangs by the door (`Lock` takes it),
whoever holds the key is inside alone, everyone else queues, and returning
the key (`Unlock`) lets the next person in. Note what this analogy makes
obvious: the mutex doesn't protect the *room* by magic — it only works if
**everyone agrees to take the key first**. One goroutine touching the data
without `Lock` and the whole guarantee evaporates (and `-race` will tell
you).

Remember demo 03 from Module 1 — `count++` is LOAD → ADD → STORE, and two
goroutines interleaving those steps lose updates. With the mutex, the
interleaving becomes impossible:

```
goroutine A: Lock ─ LOAD 5 ─ ADD ─ STORE 6 ─ Unlock
goroutine B:      Lock ──(blocked)──────────── LOAD 6 ─ ADD ─ STORE 7 ─ Unlock
```

B *cannot* start its LOAD until A's STORE is done. No lost updates, ever.

The rules, each one a classic bug when broken:

- **Unlock exactly what you locked, exactly once.** Unlocking an unlocked
  mutex panics. The idiom that makes this nearly automatic:

```go
mu.Lock()
defer mu.Unlock()   // runs on EVERY exit path — early return, panic, all of it
```

  You already know from ex9 (Module 1) what happens when a cleanup call gets
  skipped by an early `return`. The same failure with a mutex is *worse*: a
  goroutine that returns while still holding the lock doesn't just leak —
  **every future goroutine that tries to `Lock` blocks forever.** One missed
  unlock on one rare error path = the whole service seizes up minutes later.
  (That's exercise 3.)

- **A mutex protects DATA, not code.** Put the mutex next to the data it
  guards (same struct, field right above the map/counter it protects) and
  make every access path go through it. The convention you'll see in every
  real codebase — including shippio's `keyed_mutex.go`:

```go
type Inventory struct {
    mu    sync.Mutex
    stock map[string]int   // guarded by mu — nothing touches this without holding mu
}
```

- **Mutexes must not be copied after first use** (a copy is a *different*
  lock guarding nothing). Pass `*Inventory`, never `Inventory` by value.
  `go vet` catches this (`copylocks`).
- **Go mutexes are not reentrant.** The same goroutine calling `Lock` twice
  deadlocks itself — there's no "I already own it" check. Method A that
  locks and then calls method B that also locks = self-deadlock. Structure:
  exported methods lock, unexported helpers assume the lock is held.
- **Keep critical sections small.** Hold the lock for the map write, not for
  the network call before it — everything under the lock is serialized, so
  lock time is the ceiling on your concurrency. (But see exercise 2 for a
  case where holding it longer is the *point*.)

▶ **Run:** `go run ./03-sync/demo/01-mutex` — the Module-1 racy counter,
fixed live, same 100k increments.

## 2. sync.RWMutex — many readers OR one writer

A plain Mutex serializes *everyone* — including two goroutines that only
want to *read*. But two simultaneous readers can't corrupt anything; reads
only race with *writes*. `RWMutex` encodes exactly that:

```go
var mu sync.RWMutex

mu.RLock()  ... mu.RUnlock()   // READ lock: any number may hold it together
mu.Lock()   ... mu.Unlock()    // WRITE lock: exclusive — waits for all readers,
                               // blocks new ones
```

Rules of thumb:

- Reach for it when the workload is **read-heavy** (config lookups, caches
  read on every request, written rarely).
- **It is not free.** RWMutex bookkeeping is more expensive than Mutex's; on
  a write-heavy or low-contention workload it's often *slower* than a plain
  Mutex. Default to Mutex; upgrade to RWMutex only when reads dominate and
  contention is real (i.e., you measured).
- **No upgrade path.** You cannot turn an RLock you hold into a Lock —
  trying to acquire `Lock` while holding `RLock` deadlocks (the writer waits
  for readers to leave; you *are* a reader who's waiting on the writer).
  Release the read lock first, take the write lock, and **re-check the
  condition** — the world may have changed in the gap.

▶ **Run:** `go run ./03-sync/demo/02-rwmutex` — 50 concurrent readers timed
under Mutex vs RWMutex; watch reads serialize under one and overlap under
the other.

## 3. sync/atomic — hardware-level indivisible operations

For the special case where the shared thing is a single number or flag, the
CPU itself offers indivisible read-modify-write instructions — no lock
needed:

```go
var count atomic.Int64

count.Add(1)          // the whole LOAD+ADD+STORE as ONE indivisible step
v := count.Load()     // safe read
count.Store(42)       // safe write
swapped := count.CompareAndSwap(41, 42)  // "set to 42 IF currently 41" — atomically
```

Use the typed API (`atomic.Int64`, `atomic.Bool`, `atomic.Pointer[T]`) —
the older function style (`atomic.AddInt64(&n, 1)`) works but is easier to
misuse.

- **When:** counters, gauges, flags, sequence numbers — one word of state,
  simple transitions. This is the fastest option under contention (no
  parking, no scheduler involvement).
- **When NOT:** the moment two pieces of state must change *together* (map +
  count, check-then-update logic spanning fields). Atomics make ONE
  operation indivisible; they cannot make two operations atomic *together* —
  that's what mutexes are for. Composing multiple atomics to fake a lock is
  a classic source of subtle bugs; don't.

▶ **Run:** `go run ./03-sync/demo/03-atomic` — the same counter three ways
(racy, mutex, atomic) with rough timings side by side.

## 4. sync.Once — "exactly once, and everyone waits"

The lazy-initialization problem: load the config file the first time anyone
asks, exactly once, even if 100 goroutines ask simultaneously.

```go
var (
    once   sync.Once
    config *Config
)

func GetConfig() *Config {
    once.Do(loadConfig)   // first caller runs it; everyone else WAITS, then proceeds
    return config          // guaranteed: loadConfig fully finished before any return
}
```

Two separate guarantees, and interviews probe both:

1. **The function runs exactly once**, no matter how many goroutines call
   `Do` concurrently or how many times.
2. **Nobody returns from `Do` until the function has completed.** The losers
   don't skip ahead while the winner is still loading — they *block* until
   initialization is done, then proceed. (Without this, goroutine B could
   see `config == nil` while A is still mid-load.)

The naive version — `if !done { done = true; load() }` — is a **data race**
on `done` AND allows exactly the B-sees-nil bug above (check-then-act, the
same TOCTOU shape as exercise 2). You'll build a correct `Once` yourself in
exercise 4; it takes a mutex and more care than it looks.

Also in the family (Go 1.21+): `sync.OnceFunc`, `sync.OnceValue` — wrappers
that turn a function into a call-once function. Worth knowing they exist.

▶ **Run:** `go run ./03-sync/demo/04-once` — 20 goroutines race to
initialize; watch "initializing..." print once while all 20 get the result.

## 5. Honorable mentions: sync.Map and sync.Cond

**`sync.Map`** — sounds like "the concurrent map you should use"; it isn't.
Its own docs steer you away: it's optimized for two narrow patterns (keys
written once then read many times; disjoint key sets per goroutine). For the
common write-heavy / overlapping-keys case, a plain `map` + `Mutex` is
simpler AND usually faster. Interview-ready line: *"I'd start with map+Mutex
and only consider sync.Map if profiling shows lock contention on a
read-mostly map."* (This confirms what the Module-1 sharding notes said.)

**`sync.Cond`** — "wait until some condition becomes true, without
busy-polling" (a queue becomes non-empty, a state flips). Powerful but
subtle (spurious wakeups, `for !condition { cond.Wait() }` loops), and in
Go, **a channel usually expresses the same thing more clearly** — close a
channel to broadcast "condition now true" (you saw `close(done)` as a
broadcast in Drill 10). Know it exists, recognize it in code; reach for
channels first. We won't drill it.

## 6. The memory model — why unsynchronized code has NO guarantees

This is the theory underneath every race you've fixed. It's also the
interview question that separates "used goroutines" from "understands them."

**The uncomfortable truth:** without synchronization, there is no guarantee
that a write made by goroutine A is *ever* observed by goroutine B — nor any
guarantee about *order*. This loop may never terminate:

```go
var done bool
go func() { done = true }()
for !done { }   // may spin FOREVER — no sync means no visibility guarantee
```

Why? Two layers conspire:

- **The compiler** may keep `done` in a register, hoist the check out of the
  loop, reorder writes that "don't depend on each other."
- **The CPU** has per-core caches and store buffers; a write sits in core
  1's buffer, invisible to core 2, for an unbounded time. Cores may also
  observe each other's writes out of order.

Both are legal *precisely because* the code has a data race. The Go memory
model's contract is one sentence long, and it's the sentence that matters:

> **Programs without data races behave as if everything ran in one simple
> interleaved order. Programs WITH data races have no guarantees at all.**

So "it's just a flag, a torn read can't hurt" is never valid reasoning —
racy code isn't "slightly wrong," it's outside the contract entirely.

### happens-before: the formal name for "safe"

Synchronization primitives don't just block goroutines — they **publish
memory**. Each one creates a *happens-before* edge: everything A did before
the sync point is guaranteed visible to B after it. The edges you now own:

| A does ... | B does ... | everything before A's action is visible to B |
|---|---|---|
| `ch <- v` (send) | `<-ch` receives it | ✓ (why channel handoffs never race) |
| `close(ch)` | receive sees closed | ✓ (why `close(done)` broadcasts safely) |
| `mu.Unlock()` | later `mu.Lock()` | ✓ (why mutex-guarded data is safe) |
| `wg.Done()` | `wg.Wait()` returns | ✓ (why merge-after-Wait needs no locks!) |
| `once.Do(f)` returns anywhere | — | everything f wrote is visible ✓ |

That fourth row is the formal reason your Module-1 sharding pattern
(`WordFrequency`, `TotalSize`) is correct with zero locks: every worker's
writes happen-before its `Done()`, which happens-before `Wait()` returning,
which happens-before the merge loop reading. You built happens-before
chains for two modules without the vocabulary; now you have it.

## 7. The decision framework: channel, mutex, atomic, or sharding?

The interview form of this question: *"when do you use channels vs
mutexes?"* Your framework, tool by tool:

- **Sharding (no sharing at all)** — batch job with a clear
  fan-out/merge boundary: workers fill private slots, merge after `Wait()`.
  Zero contention. Doesn't fit long-lived always-shared state.
- **Channel** — when data/ownership *moves*: pipelines, work queues,
  results streaming, "first wins", completion signals. The value leaves one
  goroutine and belongs to another.
- **Mutex** — when state *stays put* and many goroutines visit it: caches,
  registries, counters-with-invariants, anything check-then-update. Rule of
  thumb from the Go wiki itself: channels for passing ownership, **mutexes
  for protecting shared state**. Wrapping a mutex-shaped problem in
  channels gives you more code and less clarity.
- **Atomic** — mutex's special case: state is ONE word, transitions are
  simple (add, swap, flag). Fastest, least flexible.

And the honest engineering order: **simplest correct thing first, measure
before optimizing.** map+Mutex before sync.Map, Mutex before RWMutex,
either before a clever lock-free scheme. This module's benchmarks (ex1)
give you the numbers to back that up.

## 8. Interview questions

1. What does `count++` compile to, and how exactly does a mutex make the
   lost-update interleaving impossible? Draw the two-goroutine timeline.
2. Why is `defer mu.Unlock()` more than style? Describe the production
   failure mode of an early `return` while holding a lock — and how it
   differs from ex9's WaitGroup version of the same mistake.
3. RWMutex: when does it beat a plain Mutex, when is it slower, and what
   happens if you try to acquire the write lock while holding a read lock?
4. What can `sync/atomic` do that a mutex also does — and what can a mutex
   do that atomics fundamentally cannot?
5. State BOTH guarantees of `sync.Once.Do`. Why is
   `if !done { done = true; f() }` broken twice over?
6. The `for !done {}` spin loop with an unsynchronized flag: name the two
   independent layers that can each prevent it from ever terminating.
7. State the Go memory model's contract in one sentence. What does
   "happens-before" mean, and which edge makes the shard-then-merge pattern
   correct without any locks?
8. Channels vs mutexes: give the one-line rule, and one concrete example on
   each side from code you've actually written in this repo.

## 9. This module's files

```
demo/01-mutex/       the Module-1 racy counter, fixed live
demo/02-rwmutex/     50 readers: serialized vs overlapping, timed
demo/03-atomic/      counter three ways with rough timings
demo/04-once/        20 goroutines race to initialize; init runs once
exercises/           YOUR work — see exercises/README.md
```

```powershell
go run ./03-sync/demo/01-mutex              # demos
go test ./03-sync/...                       # fast native runs
.\test-race.ps1 ./03-sync/...               # definition of done
go test -bench=. -run=^$ ./03-sync/exercises   # ex1 benchmarks (after solving)
```
