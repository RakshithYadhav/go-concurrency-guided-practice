# Module 4 — Concurrency Patterns I: Pipelines & Pools

## 0. Why this module exists

You now own all three tools: goroutines (Module 1), channels (Module 2),
and locks (Module 3). This module is different. It teaches no new tools.
It teaches the **recipes** — the handful of shapes that real Go programs
are actually built from.

Here's the honest truth about production Go: you almost never invent a
new concurrency design. You recognize which of a few known shapes your
problem is, and you build that shape well. Your shippio CSV importer is
proof — its job runner is a **worker pool**, one of the two big shapes in
this module. By the end of this module you'll be able to build it from
scratch and explain every line.

The shapes in this module:

1. **The pipeline** — data flows through a chain of stages.
2. **Fan-out / fan-in** — one slow stage gets N helpers.
3. **The worker pool** — N long-lived workers share one job queue.
4. **Bounded parallelism** — doing many things at once, but never TOO
   many at once.

And one failure mode that haunts all of them: **the goroutine leak**.

## 1. The pipeline — an assembly line for data

### The idea

Remember the assembly-line picture from Module 3's notes: a worker bolts
on a wheel, slides the car to the next station, and never touches it
again. A pipeline is exactly that, in code. Each **stage** is a goroutine.
Each conveyor belt between stations is a **channel**.

A stage is just a function with this shape:

```go
func square(in <-chan int) <-chan int {
    out := make(chan int)
    go func() {
        defer close(out)
        for v := range in {
            out <- v * v
        }
    }()
    return out
}
```

Read it slowly. The function:

1. Makes its own output channel.
2. Starts one goroutine that reads everything from `in`, transforms it,
   and sends it to `out`.
3. Closes `out` when the input runs dry.
4. Returns the output channel **immediately** — the work happens in the
   background. (This is Module 2's generator pattern, reused as a
   building block.)

Chaining stages then reads like a sentence:

```go
for v := range square(generate(1, 2, 3, 4)) {
    fmt.Println(v)   // 1, 4, 9, 16
}
```

### Walk one value through, step by step

Take the number 3. Here's its whole life:

1. `generate` sends 3 into its output channel. It blocks — unbuffered
   channels are a handshake (Module 2).
2. `square`'s goroutine receives 3. The handshake completes. `generate`
   moves on to 4.
3. `square` computes 9 and sends it on ITS output channel. Now `square`
   blocks, waiting for a receiver.
4. `main`'s `range` receives 9 and prints it. Handshake completes.
5. When `generate` runs out of numbers, it closes its channel. `square`'s
   `range` loop ends, so its `defer close(out)` runs. `main`'s `range`
   loop ends. Everything shuts down cleanly, in order, like dominoes.

Notice what you did NOT need: no WaitGroup, no mutex, no counters. The
closes cascade down the chain and shut everything off.

### The two rules of pipeline hygiene

1. **The stage that sends on a channel is the one that closes it.**
   Nobody else. This is Module 2's close-ownership rule, and pipelines
   live or die by it.
2. **Every stage closes its output when its input is done.** That's what
   makes the domino shutdown work. Forget one `close` and every stage
   after it blocks forever on `range` — a leak (Section 5).

▶ **Run:** `go run ./04-patterns/demo/01-pipeline` — a 3-stage pipeline,
each value's journey printed as it happens.

### When do you WANT a pipeline?

When your problem naturally reads as "take each item, do A to it, then B,
then C" — parsing then validating then saving, downloading then resizing
then uploading. The win isn't raw speed at first. It's that every stage
runs at the same time: while stage B works on item 1, stage A is already
working on item 2. Throughput without any shared state to lock.

### What if a stage can FAIL? Carrying errors through a pipeline

Everything above moves clean values. Real stages fail: a row doesn't
parse, a fetch times out. A channel can only carry one type — so make
that type carry the error too:

```go
type parsed struct {
    row Row
    err error
}

func parse(in <-chan string) <-chan parsed {
    out := make(chan parsed)
    go func() {
        defer close(out)
        for line := range in {
            row, err := parseLine(line)
            out <- parsed{row: row, err: err}   // send BOTH, always
        }
    }()
    return out
}
```

The stage never dies on a bad item. It reports the error as data, and a
later stage (or the final consumer) decides: skip the bad row? count it?
stop everything? That decision belongs to the END of the pipeline, not
the middle — the middle just keeps the belt moving. This result-with-
error shape is everywhere in production Go; your shippio importer's
row-level errors are exactly this. ("Stop everything on first error" is
also real, and needs cancellation — that's `context` and `errgroup`,
Module 5.)

### Backpressure — the pipeline regulating its own speed

One more thing you get for free and should notice. Say `generate` is
instant but `square` takes 10ms per item. Does `generate` race ahead and
pile up 10,000 unprocessed values in memory? No — its send BLOCKS until
`square` is ready to receive. The slow stage automatically slows down
every stage behind it. That's called **backpressure**, and unbuffered
channels give it to you by default. A small buffer (`make(chan int, 64)`)
smooths out bursts, but it does not fix a stage that's slower on average —
the buffer just fills, and then everything blocks like before, having
spent memory to delay the inevitable. Module 6 goes deeper; for now, know
that "producer blocks when the consumer falls behind" is a FEATURE, and
reaching for a giant buffer to "fix" it is usually hiding the real
problem.

## 2. Fan-out / fan-in — when one stage is the bottleneck

### The problem, with numbers

Say your pipeline has three stages, and the middle one calls a slow API:
10ms per item. 100 items × 10ms = **one full second** stuck in that one
stage, while the stages before and after mostly sit idle waiting.

The fix is obvious once you say it out loud: hire more copies of the slow
worker.

- **Fan-out** = start N goroutines that all read from the SAME input
  channel. Channels are safe for many readers — each value is delivered
  to exactly ONE of them (Module 2). The N workers naturally share the
  work: whoever is free grabs the next item.
- **Fan-in** = merge the N output channels back into one, so the rest of
  the pipeline doesn't need to know the middle got parallelized. You
  already built this: Module 2's ex4 `Merge` IS fan-in — a WaitGroup
  counting the senders, and a closer goroutine that closes the merged
  channel after `Wait()`.

Same 100 items with 8 copies of the slow stage: about 130ms instead of
1000ms. That ratio — roughly N times faster until something else becomes
the bottleneck — is the whole sales pitch.

### My understanding (checked and confirmed)

The channel acts like a load balancer, distributing the incoming values
across the workers. Each of the N workers publishes its own results on
its own output channel. We create N workers, let them work, then collect
all N of their output channels and merge them into a single channel.

One precision worth keeping: it's not a REAL load balancer — no health
checks, no weighting, nothing that clever. It just hands each value to
whichever receiver happens to be free at that moment (Module 2). The
side effect is load balancing anyway: a busy worker can't be ready to
receive, so work naturally flows to whoever's idle.

### This is NOT a race — it's dividing work, not racing for it

Easy to mix this up with Module 2's `FirstResult` ("first answer wins"),
so let's be precise about the difference.

In `FirstResult`, many goroutines all try to answer the SAME question,
and you keep only whichever one finishes first — the rest are wasted
work, deliberately thrown away.

Fan-out does something completely different: every worker gets a
DIFFERENT item. Nothing is thrown away. Nothing races against anything
else for the same prize. `slowDouble` still takes 10ms per item, always —
that never changes, no matter how many workers you add. What changes is
**how many items get worked on at the same moment.**

Trace it with real numbers. 24 items, 10ms each:

- **1 worker:** does item 1 (10ms), THEN item 2 (10ms), THEN item 3
  (10ms)... nothing overlaps. 24 × 10ms = 240ms total.
- **8 workers, same input channel:** remember from Module 2 — when many
  goroutines receive from one channel, each value goes to exactly ONE of
  them. So worker 1 grabs item 1 and starts its 10ms. At the very same
  moment, worker 2 grabs item 2 and starts ITS OWN 10ms. Same for workers
  3 through 8. All eight 10ms jobs run **simultaneously**. After about
  10ms, all eight finish, and each worker immediately grabs the next
  unclaimed item. Three "rounds" of 8-at-a-time ≈ 30ms total, not 240ms.

Every item still gets processed, exactly once, by exactly one worker.
The per-item cost (10ms) never drops. What drops is the total wall-clock
time, because work that used to be stacked up one-after-another is now
happening in parallel.

### The price: order is lost

With one worker, item 1's result comes out before item 2's. With eight
workers racing, results come out **in completion order, not input
order**. Sometimes that's fine (counting, summing). Sometimes it's not
(writing lines back to a file in order). If order matters, you tag each
item with its index and reassemble at the end — Module 1's slots pattern
coming back. Know the price before you pay it.

▶ **Run:** `go run ./04-patterns/demo/02-fanout` — the same slow stage
timed twice: serial, then fanned out 8 wide. Watch the clock and watch
the output order scramble.

## 3. The worker pool — a kitchen with N cooks

### The picture

A restaurant kitchen. Orders come in and get clipped to a ticket rail.
Three cooks stand at the rail. Whoever is free pulls the next ticket,
cooks it, puts the plate on the pass, and pulls the next ticket. Orders
never wait for a SPECIFIC cook — they wait for ANY free cook.

In code:

- The ticket rail = a `jobs` channel.
- The cooks = N goroutines, all doing `for job := range jobs`.
- The pass = a `results` channel they all send to.

```go
func worker(jobs <-chan Job, results chan<- Result, wg *sync.WaitGroup) {
    defer wg.Done()
    for j := range jobs {
        results <- process(j)
    }
}
```

Somebody closes `jobs` when there's no more work — that's what ends every
worker's `range` loop and lets the cooks go home. And because N workers
all send on `results`, closing IT needs the multi-sender pattern you
learned in Module 2: a WaitGroup plus one closer goroutine.

### Why the closer goroutine can't just be `wg.Wait(); close(results)` in main

Easy to think "why not skip the extra goroutine and just do this
straight-line, right before the receive loop?"

```go
wg.Wait()          // WRONG: wait for all workers first...
close(results)     // ...then close...
for r := range results { ... }   // ...then finally start receiving
```

Trace what happens. A worker finishes a job and sends on `results`
(unbuffered — the send blocks until someone is receiving RIGHT NOW). But
nobody's receiving: `main` is stuck on `wg.Wait()`, which runs BEFORE the
receive loop in this version. The worker can't finish its send, so it
can never reach `wg.Done()`. Since `wg.Wait()` needs every worker to call
`Done()`, and none of them can, `wg.Wait()` never returns either. Every
goroutine is now frozen, each waiting on something only another frozen
goroutine could unblock. Deadlock.

The real code avoids this by running `wg.Wait()` in its own goroutine, so
`main` doesn't wait for anything — it falls straight through to the
receive loop immediately:

```go
go func() {
    wg.Wait()
    close(results)
}()
for r := range results { ... }   // main starts receiving right away
```

Now `main` is actively receiving from the very start, so worker sends
never block on a missing receiver. Meanwhile the side goroutine quietly
waits for all workers to finish, then closes `results` — which is what
ends `main`'s `range` loop.

**The rule:** `wg.Wait()` and the actual receiving have to run **at the
same time**, not one after the other. Senders need an active receiver
*while they're still working*, not only after they're already done.
That's the whole reason the wait-and-close needs its own goroutine.

### Wait — isn't this just fan-out?

Almost. Same machinery: N goroutines sharing one input channel. The
difference is framing and lifetime:

- **Fan-out** is a pipeline detail: "this one stage is slow, widen it."
  The copies live only as long as that pipeline run.
- **A worker pool** is a standing service: it starts when your program
  starts, sits there, and chews through whatever lands on the queue —
  your shippio runner processing import jobs, a server handling uploads.

If you understand one, you understand the other. Interviewers ask for
"a worker pool" by name, so own the name.

### How many workers?

There's no magic number, but there is a decision rule:

- Work that waits on the network or disk (**I/O-bound**): more workers
  than CPUs is fine — dozens, even hundreds. They spend most of their
  time blocked, not computing.
- Work that burns CPU (**CPU-bound**): more workers than CPU cores just
  makes them fight over the cores. `runtime.NumCPU()` is the honest
  starting point.
- Either way: make it a parameter, measure, adjust. Never hardcode a
  guess and walk away.

### Stopping a pool — there are two different "stops"

Production pools stop in two distinct ways, and mixing them up causes
either lost work or hung shutdowns:

1. **Drain and stop** ("finish what's queued, then go home"): stop
   feeding the jobs channel and `close` it. Workers finish everything
   already on the rail, their `range` loops end, done. Nothing is lost.
   This is the normal, graceful path.
2. **Stop NOW** ("abandon the queue"): closing `jobs` can't do this —
   workers would still drain it. You need a stop signal the workers check
   between (or during) jobs: the done channel from Section 5, or, in
   Module 5, a `context`. Your shippio runner's `Shutdown` is shape 1
   with a timeout that falls back to shape 2 — that design will make
   full sense by end of Module 5.

One production landmine while we're here: **a panic in one worker kills
the entire program**, not just that worker — an unrecovered panic tears
down the whole process no matter which goroutine it starts in. Pools
that run OTHER people's job code usually put a `defer`ed `recover` at
the top of the worker loop, turning "one poisoned job" into "one failed
result" instead of an outage. You don't need to build that today — just
know the failure mode exists and where the fix goes.

▶ **Run:** `go run ./04-patterns/demo/03-workerpool` — 3 workers, 9 jobs.
The printout shows which worker grabbed which job: never the same worker
twice in a row by design, just whoever was free.

## 4. Bounded parallelism — "all at once" is a bug

### The trap

Module 1 taught you `go` is cheap, and it is. So the tempting shape is:

```go
for _, url := range urls {
    go fetch(url)          // 500,000 urls = 500,000 goroutines
}
```

Cheap is not free. Half a million in-flight fetches means half a million
open connections, half a million buffers in memory — and the site you're
fetching from sees something that looks exactly like an attack. Unbounded
concurrency doesn't fail on your laptop with 10 test items. It fails in
production with real volume. That's what makes it nasty.

### The rule and the two shapes

**Decide the maximum number of things happening at once. Enforce it.**

Two standard ways, both of which you now have the parts for:

1. **A worker pool** (Section 3): N workers = at most N things in flight.
   The bound falls out of the design for free.
2. **A semaphore**: keep the one-goroutine-per-item shape, but make each
   goroutine acquire a slot before working. A buffered channel IS the
   semaphore — capacity N, send to take a slot, receive to give it back:

```go
sem := make(chan struct{}, 8)     // 8 slots, no more
for _, url := range urls {
    go func(u string) {
        sem <- struct{}{}          // take a slot (blocks if all 8 busy)
        defer func() { <-sem }()   // give it back, no matter what
        fetch(u)
    }(url)
}
```

Why does this bound anything? The channel holds at most 8 tokens. The 9th
goroutine's send blocks until someone receives — that is, until a running
goroutine finishes and frees a slot. The buffer's capacity IS the limit.
(Module 6 formalizes this and adds `x/sync/semaphore`; the buffered-chan
version is the one to know cold.)

## 5. Goroutine leaks — the pattern's failure mode

### How a pipeline leaks

Every pattern above has goroutines sending into channels. Now ask the
Module 1 golden-rule question: **how does each of them stop?** Answer:
only when a receiver takes their value, or their input closes. So what
happens if the consumer walks away early?

```go
func firstMatch(items []string) string {
    out := make(chan string)
    go func() {
        defer close(out)
        for _, it := range items {
            if matches(it) {
                out <- it          // second match: blocks FOREVER
            }
        }
    }()
    return <-out                   // takes ONE value, returns
}
```

Walk it through: the consumer takes the first match and returns. The
producer finds a second match and sends. Nobody will ever receive it.
That goroutine is now blocked on line `out <- it` until the program
dies. It's not garbage-collected — Go never collects a blocked
goroutine. Call this function in a server handling 50 requests a second,
and you leak up to 50 goroutines a second, each pinning its memory.
Sound familiar? It's Module 1 ex4's leak wearing pipeline clothes.

### A confusion worth heading off: `<-out` is ONE receive, not a standing listener

Easy trap: looking at `return <-out` and picturing it as something that
keeps running, the way the producer's loop keeps running. It doesn't.

`<-out` means "take exactly one value out of this channel, whenever one
shows up, then stop." It is not a subscription. It is not
`for v := range out`, which WOULD keep receiving, over and over, until
the channel closes. `<-out` is a single action that happens once.

Picture a mailbox. You walk up, and you're willing to stand there and
wait until exactly one letter shows up. The moment one arrives, you take
it and walk away. You are now gone. Nobody is standing at that mailbox
anymore. If a second letter gets dropped in five minutes later, it just
sits there — you already left, and you were the only person who was
ever going to check that particular mailbox.

`return <-out` is that one trip to the mailbox. It runs once, it gets
the FIRST match, and then `firstMatch` returns — done, finished, no more
code left to run. There is no line anywhere in this function that will
ever call `<-out` again. Not "not right now" — never again, for the rest
of the program's life. So when the producer later tries to send its
SECOND match, there is no receiver waiting for it, not temporarily, but
permanently. That is the entire reason the send blocks forever instead
of just waiting a while.

### The done-channel fix

The sender needs a second exit. That's `select` (Module 2) plus one extra
channel whose only job is to say "stop":

```go
func firstMatch(items []string) string {
    out := make(chan string)
    done := make(chan struct{})
    defer close(done)              // runs when firstMatch returns

    go func() {
        defer close(out)
        for _, it := range items {
            if matches(it) {
                select {
                case out <- it:    // normal path: someone received
                case <-done:       // consumer left: stop, exit, clean
                    return
                }
            }
        }
    }()
    return <-out
}
```

Now trace the leak case again: consumer takes the first value and
returns. `defer close(done)` fires. The producer's `select` sees `done`
closed — remember from Module 2, receiving from a closed channel never
blocks — takes that case, returns, and the goroutine ends. No leak.

Two details worth saying out loud:

- `close(done)` is a **broadcast**: every goroutine selecting on it gets
  unblocked at once, no matter how many there are (Module 2's Drill 10
  idea, now doing real work).
- The pattern generalizes: **every send in a cancellable pipeline is a
  `select` between the send and the stop signal.** In Module 5 the
  hand-rolled `done` channel becomes `ctx.Done()` — same idea, standard
  spelling, plus timeouts for free.

▶ **Run:** `go run ./04-patterns/demo/04-leak` — the leaky version called
50 times (watch the goroutine count climb and stay up), then the fixed
version (watch it stay flat).

## 6. Interview questions

Answer these out loud at review time. Precise beats fast.

1. Write the standard pipeline-stage shape from memory. Who closes what,
   and what happens — stage by stage — when the first channel closes?
2. Fan-out and fan-in: define both, explain what property of channels
   makes fan-out safe with zero extra code, and name the price you pay
   in output ordering. Where did you already build fan-in in this repo?
3. Worker pool: what are the three moving parts, who closes the jobs
   channel, and why does closing the results channel need a WaitGroup
   plus a closer goroutine? How do you pick N for I/O-bound vs CPU-bound
   work?
4. A teammate writes `for _, x := range items { go handle(x) }` over an
   unbounded input. Explain what breaks at scale and give BOTH standard
   fixes, including why a buffered channel enforces a limit at all.
5. Show, with the two-goroutine trace, how "consumer returns after one
   value" leaks the producer. Then write the done-channel fix and explain
   why receiving on a closed channel makes it work as a broadcast.
6. Why does a pipeline need no mutex anywhere, even though many
   goroutines are involved? (Hint: Module 3's ownership framing —
   who owns a value at any given moment?)
7. A pipeline stage can fail per-item. How do errors travel through a
   channel-based pipeline, and WHERE should the skip-vs-stop decision
   live? What extra machinery does "stop everything on first error" need?
8. What is backpressure, which channel type gives it to you by default,
   and why is a giant buffer usually the wrong fix for a slow consumer?
   Bonus: what are the two different ways a worker pool can stop, and
   which one does closing the jobs channel give you?

## 7. This module's files

```
demo/01-pipeline/    3-stage pipeline, one value's journey narrated
demo/02-fanout/      the slow stage: serial vs 8-wide, timed
demo/03-workerpool/  3 cooks, 9 tickets, who grabbed what
demo/04-leak/        the leak measured live, then the fix, also measured
exercises/           YOUR work — see exercises/README.md
```

```powershell
go run ./04-patterns/demo/01-pipeline       # demos
go test ./04-patterns/...                   # fast native runs
.\test-race.ps1 ./04-patterns/...           # definition of done
```
