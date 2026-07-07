# Module 3 — Locks, Atomics, and the Memory Model

*Read this top to bottom. Run each demo when you see ▶ Run. You need
Modules 1 and 2 first: goroutines, WaitGroup, closures, races, channels.*

---

## 0. Why this module exists

So far you know two safe ways for goroutines to work with data:

1. **Give each goroutine its own slot.** Nobody shares anything. You merge
   the results after `wg.Wait()`. This was Module 1.
2. **Pass values through a channel.** The data moves from one goroutine to
   another. Only one goroutine holds it at a time. This was Module 2.

Notice what both have in common: **they avoid sharing.** No two goroutines
ever touch the same data at the same time. That's why they're safe.

But sometimes you can't avoid sharing. Sometimes the program really needs
**one shared thing that many goroutines use at once**, for as long as the
program runs. For example:

- A **cache**. Every request reads from it. Some requests write to it.
- A **counter**. Every request adds 1 to it.
- A **config**. It loads once, then everyone reads it.

You *could* force these into channels or the slot-merge pattern. But the
code gets long, ugly, and often slower. There's a simpler tool for this
case: **keep the shared thing, and make goroutines take turns using it.**
The thing that makes them take turns is called a **lock**. That's what this
module teaches.

By the end you'll also know *when* to pick each tool — lock, channel, or
no-sharing. Interviewers love that question.

## 1. sync.Mutex — the lock

**Mutex** is short for **MUT**ual **EX**clusion. It's a lock. When one
goroutine holds the lock, every other goroutine that wants it must wait.

```go
var mu sync.Mutex

mu.Lock()     // wait until the lock is free, then take it
count++       // only ONE goroutine can be on this line at a time
mu.Unlock()   // give the lock back; the next waiter can now take it
```

The code between `Lock()` and `Unlock()` is called the **critical
section**. Only one goroutine can be inside it at a time.

**Analogy: a restroom with one key.** The key hangs by the door. To go in,
you take the key (`Lock`). While you have the key, nobody else can enter.
When you come out, you hang the key back (`Unlock`), and the next person
in line takes it. Notice something important: the key doesn't protect the
room by magic. It only works because **everyone agrees to take the key
first**. If one goroutine touches the data without calling `Lock`, the
protection is gone. (And `-race` will catch it.)

Why does this fix the counter bug from Module 1? Remember: `count++` is
really three steps — LOAD, ADD, STORE. Two goroutines could interleave
those steps and lose updates. With a mutex, they can't:

```
goroutine A: Lock ─ LOAD 5 ─ ADD ─ STORE 6 ─ Unlock
goroutine B:      Lock ──(waiting...)──────── LOAD 6 ─ ADD ─ STORE 7 ─ Unlock
```

B can't even start its LOAD until A is completely done. No lost updates.
Ever.

### The rules (each one is a classic bug when broken)

**Rule 1: always unlock, on every path out of the function.** The idiom:

```go
mu.Lock()
defer mu.Unlock()   // runs no matter how the function exits
```

You saw in ex9 (Module 1) what happens when an early `return` skips a
cleanup call. With a mutex it's worse. If a goroutine returns while still
holding the lock, the lock is never given back. **Every future goroutine
that calls `Lock` waits forever.** One missed unlock on one rare error path
can freeze the whole service — minutes later, far from the actual bug.
That's exercise 3.

**Rule 2: a mutex protects data, not code.** Put the mutex right next to
the data it guards — same struct, the field just above it:

```go
type Inventory struct {
    mu    sync.Mutex
    stock map[string]int   // guarded by mu — never touch this without holding mu
}
```

This is the convention in every real Go codebase, including your shippio
project's `keyed_mutex.go`.

**Rule 3: never copy a mutex.** A copy is a *different* lock. It guards
nothing. Pass `*Inventory` (a pointer), never `Inventory` by value.
`go vet` catches this mistake.

**Rule 4: don't lock twice from the same goroutine.** Go mutexes have no
"I already own it" check. If method A takes the lock and then calls method
B, and B also tries to take the same lock — that goroutine is now waiting
for itself. Forever. The usual structure: public methods take the lock;
private helper methods assume it's already held.

**Rule 5: keep the locked section short.** Everything inside the lock runs
one-goroutine-at-a-time. Hold the lock while you write to the map. Don't
hold it while you make a slow network call — unless you have a reason
(exercise 2 has one).

▶ **Run:** `go run ./03-sync/demo/01-mutex` — the racy counter from
Module 1, fixed live.

## 2. sync.RWMutex — many readers, or one writer

A plain Mutex makes *everyone* take turns — even two goroutines that only
want to **read**. That's wasted waiting. Two readers can't corrupt
anything. Reads only clash with **writes**.

`RWMutex` has two kinds of lock:

```go
var mu sync.RWMutex

mu.RLock()  ... mu.RUnlock()   // READ lock: many goroutines can hold this together
mu.Lock()   ... mu.Unlock()    // WRITE lock: fully exclusive, same as a Mutex
```

Any number of readers can hold the read lock at the same time. A writer
must wait for all of them to leave, and while a writer is inside, nobody
else gets in.

When to use it:

- Use it when reads **vastly outnumber** writes. Example: a config that
  every request reads, but that changes once an hour.
- **It's not free.** RWMutex does more bookkeeping than Mutex. If writes
  are common, or there's little contention, RWMutex is often *slower*.
  Start with a plain Mutex. Upgrade only if you measured a real win.
- **You can't upgrade a read lock into a write lock.** If you hold `RLock`
  and try to take `Lock`, you deadlock: the writer side waits for all
  readers to leave, and you *are* a reader who's now waiting for the
  writer. Release the read lock first. Then take the write lock. Then
  **re-check your condition** — the data may have changed in between.

▶ **Run:** `go run ./03-sync/demo/02-rwmutex` — 50 readers, timed. Under
Mutex they go one at a time. Under RWMutex they all go at once.

## 3. sync/atomic — the counter special-case

If the shared thing is just **one number or one flag**, you don't even
need a lock. The CPU has special instructions that do read-modify-write
as **one single, uninterruptible step**:

```go
var count atomic.Int64

count.Add(1)          // LOAD + ADD + STORE as one indivisible step
v := count.Load()     // safe read
count.Store(42)       // safe write
ok := count.CompareAndSwap(41, 42)  // "if it's 41, make it 42" — one step
```

Use the typed versions (`atomic.Int64`, `atomic.Bool`,
`atomic.Pointer[T]`). The older style (`atomic.AddInt64(&n, 1)`) works but
is easier to get wrong.

**When to use atomics:** counters, flags, IDs. One value, simple updates.
It's the fastest option — no waiting, no scheduler.

**When NOT to use them:** the moment **two things must change together**.
Say you update a map *and* a count of its entries. Atomics can make each
one safe alone, but not both together — another goroutine can slip in
between them. Keeping two things consistent is a job for a mutex. Trying
to fake a lock out of several atomics is a classic source of subtle bugs.
Don't.

▶ **Run:** `go run ./03-sync/demo/03-atomic` — the same counter three ways
(racy, mutex, atomic) with rough timings.

## 4. sync.Once — run this exactly once, and make everyone wait

The problem: load a config file the first time anyone asks for it. Only
once — even if 100 goroutines ask at the same moment.

```go
var (
    once   sync.Once
    config *Config
)

func GetConfig() *Config {
    once.Do(loadConfig)
    return config
}
```

`once.Do(f)` gives you **two** promises. Interviews ask about both:

1. **f runs exactly once.** No matter how many goroutines call `Do`, or
   how many times. The first caller runs it. That's it.
2. **Nobody returns from `Do` until f has finished.** This is the one
   people forget. The 99 "losers" don't skip ahead while the winner is
   still loading the config. They **wait**. Without this promise, a loser
   could reach `return config` while `config` is still nil.

Why can't you just write this?

```go
if !done {          // BROKEN
    done = true
    loadConfig()
}
```

Two reasons. First, it's a data race on `done` — two goroutines can both
read `false` before either writes `true`, so it runs twice. Second, even
if the race magically didn't happen, a goroutine that sees `done == true`
sails right past while the first goroutine is still *inside*
`loadConfig()` — and reads a half-built config. You'll build a correct
version yourself in exercise 4. It's harder than it looks.

Related, worth knowing they exist: `sync.OnceFunc` and `sync.OnceValue`
(Go 1.21+) wrap a function so it can only run once.

▶ **Run:** `go run ./03-sync/demo/04-once` — 20 goroutines race to load
the config. "loading config" prints once. All 20 get a result.

## 5. Two you should know about, but rarely use

**`sync.Map`** — sounds like "the map you should use for concurrency."
It isn't. Its own documentation says it's built for two narrow cases:
keys that are written once and read many times, or goroutines that each
use their own separate keys. For a normal map with mixed reads and writes,
a plain `map` plus a `Mutex` is simpler AND usually faster. The
interview-ready answer: *"I'd use map+Mutex first, and only look at
sync.Map if profiling showed lock contention on a read-mostly map."*

**`sync.Cond`** — lets goroutines sleep until some condition becomes true
("the queue has items now"). It's powerful but easy to get wrong, and in
Go a channel usually says the same thing more clearly. You've already seen
the channel version: `close(done)` broadcasts "it's ready" to every
waiter (Drill 10). Recognize `Cond` when you see it. Reach for channels
first. We won't drill it.

## 6. The memory model — why racy code has NO guarantees

This is the theory under every race you've fixed. It's also the interview
question that separates "has used goroutines" from "understands them."

Here's the uncomfortable part. Without synchronization, Go does not
promise that a write made by goroutine A will *ever* be seen by goroutine
B. This loop can spin **forever**:

```go
var done bool
go func() { done = true }()
for !done { }   // may never end
```

How is that possible? Two layers, working against you:

- **The compiler** is allowed to optimize. It may keep `done` in a CPU
  register and never re-read it from memory. From the loop's point of
  view, `done` never changes.
- **The CPU** has a separate cache per core. Core 1's write can sit in
  its own buffer, invisible to core 2, for an unbounded amount of time.

Both behaviors are legal — *because the code has a data race*. The Go
memory model's deal is one sentence:

> **No data races? Then your program behaves as if everything ran in one
> simple, interleaved order. Data races? All bets are off.**

So the excuse "it's just a flag, a stale read can't hurt" is never valid.
Racy code isn't "slightly wrong." It's outside the rules entirely.

### happens-before: the guarantee that sync tools give you

Locks and channels don't just make goroutines wait. They also **publish
memory**. Each one creates what the spec calls a *happens-before* edge:
everything goroutine A did **before** the sync point is guaranteed visible
to goroutine B **after** it.

The edges you already own:

| A does... | B does... | Then B sees everything A did before it |
|---|---|---|
| `ch <- v` (send) | receives that value | ✓ — why channel handoffs never race |
| `close(ch)` | sees the channel closed | ✓ — why `close(done)` is a safe broadcast |
| `mu.Unlock()` | takes the lock next | ✓ — why mutex-guarded data is safe |
| `wg.Done()` | `wg.Wait()` returns | ✓ — see below |
| `once.Do(f)` returns | (anyone) | ✓ — everything f wrote is visible |

Look at the fourth row. That's the formal reason your Module 1 pattern —
fill private slots, then merge after `Wait()` — needed **no locks at
all**. Each worker's writes happen-before its `Done()`. All the `Done()`
calls happen-before `Wait()` returning. And `Wait()` returning
happens-before your merge loop reading. You built happens-before chains
for two whole modules without knowing the word. Now you know the word.

## 7. Which tool when? The decision framework

The interview version of this question: *"channels or mutexes?"* Your
answer, tool by tool:

- **No sharing (slots + merge)** — best for batch jobs with a clear
  start and end: fan out, work, wait, merge. Zero contention, zero locks.
  Doesn't fit state that lives forever and is always shared.
- **Channel** — best when data **moves**: pipelines, work queues,
  streaming results, "first answer wins", completion signals. The value
  leaves one goroutine and becomes another's.
- **Mutex** — best when state **stays put** and many goroutines visit
  it: caches, registries, anything with check-then-update logic. The Go
  wiki's own rule of thumb: *channels for passing ownership, mutexes for
  protecting shared state.*
- **Atomic** — the mutex's special case: exactly one number or flag,
  simple updates. Fastest, least flexible.

And the honest engineering order: **do the simplest correct thing, then
measure.** map+Mutex before sync.Map. Mutex before RWMutex. Any of those
before something clever. Exercise 1's benchmarks give you real numbers to
back this up.

## 8. Interview questions

1. What three steps does `count++` really take? Show, with a two-goroutine
   timeline, exactly how a mutex makes the lost-update interleaving
   impossible.
2. Why is `defer mu.Unlock()` more than a style choice? Describe what
   happens to a service when one error path returns while holding a lock.
   How is that worse than ex9's skipped `wg.Done()`?
3. When does RWMutex beat a plain Mutex? When is it slower? What happens
   if you try to take the write lock while holding a read lock?
4. What can atomics do that a mutex also does? What can a mutex do that
   atomics fundamentally cannot?
5. `sync.Once.Do` makes two promises. Name both. Then explain why
   `if !done { done = true; f() }` breaks each one.
6. The `for !done {}` spin loop over an unsynchronized flag: name the two
   separate layers that can each keep it spinning forever.
7. State the Go memory model's rule in one sentence. What does
   "happens-before" mean? Which happens-before edge made Module 1's
   slot-and-merge pattern safe without locks?
8. Channels or mutexes: give the one-line rule, plus one real example of
   each from code you wrote in this repo.

## 9. This module's files

```
demo/01-mutex/       the Module-1 racy counter, fixed live
demo/02-rwmutex/     50 readers: one at a time vs all at once, timed
demo/03-atomic/      counter three ways, with timings
demo/04-once/        20 goroutines race to initialize; init runs once
exercises/           YOUR work — see exercises/README.md
```

```powershell
go run ./03-sync/demo/01-mutex              # demos
go test ./03-sync/...                       # fast native runs
.\test-race.ps1 ./03-sync/...               # definition of done
go test -bench=. -run=^$ ./03-sync/exercises   # ex1 benchmarks (after solving)
```
