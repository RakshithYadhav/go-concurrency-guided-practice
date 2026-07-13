# Module 6 — Patterns II: Production Techniques

## 0. Why this module exists

Module 4 gave you the shapes: pipelines, pools, bounded parallelism.
Module 5 made them stoppable. This module is about what production
traffic does to those shapes:

- Your pool of 8 workers politely bounds *concurrency* — and still
  hammers a partner API with 900 requests per second, because each
  request is fast. You get rate-limited or blacklisted. You need a
  **rate limiter**.
- Your cache from Module 3 has one mutex. One slow fetch for MSFT blocks
  every request for AAPL, GOOG, and 500 other symbols. You flagged this
  tradeoff yourself back in ex2. The fix is a **keyed mutex** — the
  pattern from your shippio importer, rebuilt from scratch here.
- The network is flaky. A request fails once, then would have succeeded.
  Giving up immediately wastes work; retrying instantly makes outages
  worse. You need **retries with backoff and jitter**.
- Writing to the database once per row is 10,000 round trips. Writing
  once per 500 rows is 20. You need **batching** — flush by size or by
  age, whichever comes first.

Four tools. All of them are built from parts you already own: channels,
select, mutexes, timers, context. Nothing new underneath — just
production shapes on top.

## 1. Semaphores, finished properly

You built the buffered-channel semaphore in Module 4:

```go
sem := make(chan struct{}, 8)
sem <- struct{}{}          // take a slot
defer func() { <-sem }()   // give it back
```

Know that the "official" version exists too: `golang.org/x/sync/semaphore`.

```go
sem := semaphore.NewWeighted(8)
if err := sem.Acquire(ctx, 1); err != nil { return err }  // ctx-aware!
defer sem.Release(1)
```

Two things it adds over the channel version:

1. **Context-aware acquiring.** `Acquire(ctx, 1)` gives up cleanly if
   ctx dies while waiting for a slot. The channel version needs a
   `select` wrapped around the send to do the same.
2. **Weights.** `Acquire(ctx, 3)` takes three slots at once — useful
   when jobs have different costs (a 10MB upload can count as 10, a
   1MB one as 1).

If all slots are equal and you already have the channel version, keep
it. Reach for `semaphore.Weighted` when you need ctx-aware waits or
weighted costs.

## 2. Rate limiting — "not too many at once" is not "not too fast"

### The distinction people miss

A worker pool bounds **how many things happen at the same moment**. A
rate limiter bounds **how many things happen per unit of time**. They
sound alike. They are not.

Run the numbers: 8 workers, each job takes 10ms. Every worker finishes
and immediately grabs the next job. That's 8 jobs per 10ms = **800
requests per second** hitting whatever's downstream — with concurrency
perfectly bounded at 8 the whole time. If the downstream API allows 100
requests per second, your perfectly-bounded pool is 8x over the limit
and about to get you blocked.

Concurrency limits protect YOUR memory and CPU. Rate limits protect THE
OTHER SIDE. Production code often needs both at once.

### The token bucket — the standard mental model

Picture a bucket that holds up to `b` tokens. Tokens drip in at a steady
rate `r` per second. Every request must take one token to proceed; if
the bucket is empty, the request waits for the next drip.

- The **rate** `r` is the long-term speed limit: on average, never more
  than r per second.
- The **burst** `b` is the give: if you've been quiet for a while, the
  bucket fills up, and a short burst of up to b requests can go through
  instantly before the drip-wait kicks in. Real traffic is bursty;
  allowing a small burst absorbs that without violating the average.

### The standard tool: `golang.org/x/time/rate`

```go
limiter := rate.NewLimiter(rate.Limit(100), 10)   // 100/sec, burst of 10

func handle(ctx context.Context, req Request) error {
    if err := limiter.Wait(ctx); err != nil {     // take a token (or wait for one)
        return err                                 // ctx died while waiting
    }
    return callDownstream(ctx, req)
}
```

`Wait(ctx)` blocks until a token is available — and it's ctx-aware, so a
canceled request stops waiting instead of queueing forever. There's also
`Allow()` (non-blocking: "is a token free right now, yes/no?") for when
you'd rather reject than wait — that's how servers shed load.

The DIY version — a `time.Ticker` dripping into your code at fixed
intervals — is worth building once (the exercise lets you choose), but
in production code reach for `rate.Limiter`: it handles bursts, partial
tokens, and ctx for free.

▶ **Run:** `go run ./06-patterns2/demo/01-ratelimit` — the same 10 calls
unlimited vs limited to 20/sec. Watch the timestamps spread out.

## 3. The keyed mutex — one lock per key, not one lock for everything

### The problem, which you already found yourself

Module 3, exercise 2: your `PriceCache` guards its whole map with ONE
mutex, held across the slow fetch. You were asked to defend the cost and
you did: while MSFT's slow 2-second fetch holds the lock, requests for
AAPL — a completely unrelated symbol — stand in line behind it. One slow
key freezes every key. Correct, but brutal.

The obvious wish: "I want a SEPARATE lock for each symbol." That's a
keyed mutex. MSFT's lock and AAPL's lock are different locks; work on
different keys never waits on each other; two requests for the SAME key
still serialize properly. Your shippio importer does exactly this to
serialize work per shipment while different shipments proceed in
parallel.

### The shape

You can't declare locks up front — keys arrive at runtime. So: a map
from key to its mutex, and (channels-and-locks rule from Module 3) that
map itself is shared mutable state, so IT needs its own little mutex:

```go
type KeyedMutex struct {
    mu    sync.Mutex             // guards the map below, and only the map
    locks map[string]*sync.Mutex // one lock per key, created on demand
}

func (k *KeyedMutex) Lock(key string) {
    k.mu.Lock()
    m, ok := k.locks[key]
    if !ok {
        m = &sync.Mutex{}
        k.locks[key] = m
    }
    k.mu.Unlock()   // release the MAP lock before taking the KEY lock
    m.Lock()        // may block for a long time — that's fine now
}
```

Read the order carefully. The outer `k.mu` is held only for the quick
map lookup — microseconds. It is released BEFORE `m.Lock()`, which may
block for seconds. That ordering is the whole point: the expensive wait
happens on the per-key lock, where only same-key callers queue up. If
you held `k.mu` across `m.Lock()`, you'd have rebuilt the global-lock
problem you were trying to escape — every key waiting on one lock again.

### The cleanup trap (this is exercise 2)

The tempting "tidy" move is to delete the key's mutex from the map on
unlock, so the map doesn't grow forever. Deleting it SAFELY is genuinely
hard: another goroutine may be blocked on that exact mutex right now,
and if a third goroutine then finds the key missing and creates a FRESH
mutex for the same key, you have two goroutines inside the same key's
critical section at once — mutual exclusion silently broken. You'll
diagnose and fix precisely this in ex2. (Production answers: don't
delete — maps of small mutexes are cheap; or reference-count each entry
so it's only deleted when nobody's waiting. Ex2 accepts the simple fix;
the refcount is a stretch goal.)

▶ **Run:** `go run ./06-patterns2/demo/02-keyedmutex` — same key
serializes (timed), different keys overlap (timed).

## 4. Retries — with backoff, jitter, and a budget

### Why retry at all

Networks hiccup. A packet drops, a pod restarts, a database fails over.
A huge share of failures are **transient**: try the same thing 100ms
later and it works. Giving up on the first failure turns a 100ms hiccup
into a user-visible error. So: retry.

### Why NOT to retry naively

Retrying instantly, in a tight loop, is how you turn a struggling
service into a dead one. It was slow because it's overloaded — and now
every client is re-sending every request several times as fast as
before. The fixes, layered:

1. **Backoff:** wait between attempts, and wait LONGER each time —
   classically doubling: 100ms, 200ms, 400ms, 800ms. Early retries
   catch quick hiccups; later ones give a struggling service room to
   breathe instead of piling on.
2. **Jitter:** add randomness to each wait. Here's the failure mode
   without it: a service blips for one second, and 10,000 clients all
   fail *together*. With clean exponential backoff, all 10,000 retry at
   the SAME instant, in synchronized waves — each wave its own little
   outage. This is the **thundering herd**. Jitter (e.g., each wait is
   the base amount plus a random 0–50% extra) spreads the herd out into
   harmless drizzle.
3. **A budget:** retries stop. A bounded count (or a deadline) — and the
   whole loop listens to ctx, because a canceled request must stop
   retrying immediately, even mid-wait. (Module 5: the wait itself
   becomes `select { case <-time.After(d): case <-ctx.Done(): }`.)

One more production rule worth knowing: only retry what's safe to
repeat. A timed-out payment POST might have actually gone through —
blind retry double-charges. (That's idempotency, and it gets real
treatment in the capstone and Track 8.)

▶ **Run:** `go run ./06-patterns2/demo/03-retry` — an op that fails
three times then succeeds; watch the gaps grow, with jitter making them
uneven.

## 5. Batching — trade a little latency for a lot of throughput

### The problem

Your service receives 10,000 events a minute and writes each one to the
database as it arrives. That's 10,000 round trips, each with fixed
overhead (network hop, transaction, index update). The database does
10,000 small pieces of work where ~20 big ones would do.

Batching: collect items into a buffer, flush the whole buffer at once.
`INSERT` 500 rows in one statement. Your shippio importer's fan-out
commit stage does exactly this.

### The two triggers (you need both)

- **Size:** the batch hits N items → flush. This caps memory and keeps
  batches efficient under heavy traffic.
- **Age:** the oldest item has waited T milliseconds → flush whatever's
  there, even if it's one item. Without this, a quiet stream strands
  its last few items in the buffer forever — a trickle of one event per
  minute would NEVER fill a 500-item batch. Age caps latency.

Whichever trigger fires first wins, and the timer resets after every
flush. Heavy traffic → size-triggered, efficient. Light traffic →
age-triggered, still timely. The implementation is one goroutine with a
`select` over the input channel and a timer — plus a close-flush: when
the input closes, flush the partial batch, then close the output.
(You'll build exactly this in ex4.)

Related vocabulary, one sentence each: **debouncing** waits for a quiet
gap before acting (search-as-you-type fires once the user STOPS typing);
**throttling** is rate limiting by another name (act at most once per
interval). Batching collects everything; debouncing keeps only the
latest; throttling drops what's over the limit.

▶ **Run:** `go run ./06-patterns2/demo/04-batch` — an uneven producer;
watch some batches flush on size and others on age.

## 6. Channel plumbing shapes — survey level

Three named shapes you'll meet in articles and reviews. All are small
select loops; recognize them, write them when needed:

- **or-done:** Module 5's "wrap every receive in a select against
  ctx.Done()" — extracted into a reusable function that adapts a channel
  so YOU can range over it plainly, cancellation handled inside.
- **tee:** one input channel duplicated to two outputs (audit log +
  processing, same stream).
- **bridge:** a channel OF channels flattened into one continuous
  channel (a sequence of result streams, consumed as one).

Don't memorize implementations. Remember they're all "one goroutine, one
select loop, close-ownership rules from Module 2."

## 7. Interview questions

Answer these out loud at review time. Precise beats fast.

1. A worker pool of 8 already bounds concurrency. Show, with numbers,
   how it can still overwhelm a downstream API — and explain what a rate
   limiter bounds that the pool doesn't.
2. Explain the token bucket: what the rate does, what the burst does,
   and why allowing a burst at all is useful. What's the difference
   between `Wait(ctx)` and `Allow()`, and when would a server prefer
   each?
3. Keyed mutex: why is the outer map-mutex released BEFORE taking the
   per-key mutex? What exact failure comes back if you hold it across
   the key lock?
4. The delete-on-unlock cleanup in a keyed mutex silently breaks mutual
   exclusion. Walk the three-goroutine interleaving that shows two
   goroutines inside the same key's critical section.
5. Backoff without jitter caused synchronized retry waves during a real
   outage. Explain the thundering herd, why exponential backoff alone
   doesn't fix it, and what jitter changes.
6. Why must every retry wait be a select against ctx.Done() rather than
   a plain time.Sleep? What's the user-visible difference?
7. Batching: why do you need BOTH a size trigger and an age trigger?
   What breaks with only size? Only age? Where does the final
   close-flush fit?
8. Your shippio importer uses a keyed mutex per shipment and batched
   commits. For each, name the failure you'd see if it were removed.

## 8. This module's files

```
demo/01-ratelimit/    10 calls: unlimited vs 20/sec, timestamped
demo/02-keyedmutex/   same key serializes, different keys overlap
demo/03-retry/        growing, jittered gaps between attempts, live
demo/04-batch/        size-triggered and age-triggered flushes, labeled
exercises/            YOUR work — see exercises/README.md
```

```powershell
go run ./06-patterns2/demo/01-ratelimit     # demos
go test ./06-patterns2/...                  # fast native runs
.\test-race.ps1 ./06-patterns2/...          # definition of done
```
