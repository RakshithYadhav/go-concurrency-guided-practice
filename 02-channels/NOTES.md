# Module 2 — Channels

*Read top to bottom, running each demo when the text says ▶ Run. Everything
builds on Module 1 — if "goroutines run later" or "closures capture
variables" feels shaky, skim those notes again first.*

---

## 0. The problem channels solve

In Module 1, your goroutines communicated with the rest of the program in
exactly one, very limited way: each wrote into its own pre-agreed slot
(`results[i]`), and the caller read everything only after `wg.Wait()`. That
works — but notice what it *can't* do:

- The caller can't receive results **as they're produced** — it waits for
  ALL goroutines, then reads everything at once.
- Goroutines can't hand work **to each other** — there's no way for one
  goroutine to produce items that another consumes.
- Nobody can say "give me whichever result arrives **first**" (your ex4
  `FirstResult` needed exactly this, and had to reach ahead to channels).

A **channel** fixes all three: it's a typed pipe through which one goroutine
can send values and another can receive them, **safely, with synchronization
built in**. No mutex, no slots, no races — the handoff itself is the
synchronization.

Go's design philosophy, which you'll see quoted everywhere:

> **"Don't communicate by sharing memory; share memory by communicating."**

Module 1 was the first half of that sentence (shared slices, careful
disjoint writes). This module is the second half.

## 1. What a channel is

```go
ch := make(chan int)      // a channel that carries ints
ch <- 42                  // SEND the value 42 into the channel
x := <-ch                 // RECEIVE a value from the channel into x
```

Mental model: a channel is a **conveyor belt between goroutines** with a
type stamped on it (`chan int` only carries ints). One side puts values on
(`ch <- v`), the other takes them off (`<-ch`). The runtime handles all the
locking, waking, and memory-visibility guarantees internally — a value sent
before a receive is *guaranteed* visible to the receiver, no `-race` report,
ever, for the handoff itself.

Three operations, and (Module 1 flashback) one golden question for each:
**does it block, and until when?** That question is 90% of channel mastery.

## 2. Unbuffered channels: the handshake

`make(chan T)` — no second argument — creates an **unbuffered** channel. It
has **zero storage**. A value can't sit "in" the channel; it can only pass
**directly from a sender's hand to a receiver's hand**. That means:

- **Send (`ch <- v`) blocks** until another goroutine is ready to receive.
- **Receive (`<-ch`) blocks** until another goroutine sends.

Both parties must show up; whoever arrives first **waits** for the other.
It's a handshake, not a mailbox:

```
goroutine A:   ch <- 42 ──── blocked, waiting for a receiver...
goroutine B:                                        x := <-ch
                              ────► handoff happens HERE, both continue ────►
```

▶ **Run:** `go run ./02-channels/demo/01-unbuffered` — timestamps show the
sender genuinely frozen until the receiver shows up 2 seconds later.

This blocking is a *feature*, not a limitation. You already exploited it in
Module 1's ex9 test (`done <- ...` / `<-done` to detect a hang): the receive
completing **proves** the sending goroutine got that far. An unbuffered send
is synchronization — that's why your `FirstResult` fix couldn't just be
ignored by the runtime: every send had to find a receiver or wait forever.

And that's also the danger. In ex4 you saw what happens when a send can
never find its receiver: **the goroutine blocks forever — a leak.** If NO
goroutine can proceed at all, the runtime detects it and crashes with
`fatal error: all goroutines are asleep - deadlock!`. (It only catches the
*everyone*-stuck case — a few leaked goroutines among live ones, like ex4,
go undetected. That's why leaks are sneakier than deadlocks.)

## 3. Buffered channels: the mailbox

`make(chan T, n)` creates a **buffered** channel with `n` slots of storage.
Now it *is* a mailbox:

- **Send blocks only when the buffer is FULL.** Otherwise it drops the value
  in a slot and continues immediately — no receiver needed at that moment.
- **Receive blocks only when the buffer is EMPTY.** Otherwise it takes the
  oldest value (FIFO — first in, first out) and continues.

```go
ch := make(chan int, 2)
ch <- 1        // buffer [1]       — returns immediately
ch <- 2        // buffer [1 2]     — returns immediately
ch <- 3        // buffer FULL      — BLOCKS until someone receives
```

▶ **Run:** `go run ./02-channels/demo/02-buffered` and watch exactly where
the sender stalls.

This is precisely why your one-line ex4 fix worked: `make(chan string,
len(queries))` gave every replica goroutine a guaranteed slot, so every send
completes immediately even though only the first value is ever received —
the losers deposit their result and **exit** instead of blocking forever.

Two rules of thumb real engineers use:

- **Default to unbuffered.** You get the strongest guarantee (receipt =
  proof of handoff) and the most predictable behavior.
- **Buffer with a reason you can say out loud.** Good reasons: "N producers
  each send exactly once and must not block" (ex4), "queue of size N as
  deliberate backpressure" (shippio's job queue — remember `make(chan Job,
  buffer)` in its Runner). Bad reason: "the deadlock went away when I added
  a buffer" — that's hiding a structural bug behind storage, and it comes
  back the day the buffer fills.

## 4. close, comma-ok, and range

A sender can announce "no more values, ever" by closing the channel:

```go
close(ch)
```

**Why close exists:** without it, a receiver doing `<-ch` would block
forever once the values run out — it has no way to know "done" from
"slow". Close is that signal.

Receiving from a channel actually returns **two** values:

```go
v, ok := <-ch   // the "comma-ok" form
```

- Channel open, value available → `v` = the value, `ok` = `true`
- Channel **closed and drained** → `v` = the **zero value** of the type,
  `ok` = `false` — immediately, no blocking

That second behavior is load-bearing: a closed channel **never blocks a
receiver again**. It hands out zero values with `ok=false` forever. (And
note the subtlety: close doesn't destroy buffered values — receivers first
drain whatever's still in the buffer with `ok=true`, and only *then* start
getting `ok=false`.)

`range` packages the comma-ok loop for you:

```go
for v := range ch {   // receive until the channel is CLOSED and drained
    fmt.Println(v)
}
// loop exits here only because someone closed ch
```

**What this is actually shorthand for** — and this is the detail that
matters, not just syntax sugar:

```go
for {
    v, ok := <-ch   // the comma-ok form
    if !ok {
        break        // channel is closed AND drained — stop looping
    }
    fmt.Println(v)
}
```

`range` isn't just "call `<-ch` repeatedly" — it's "call `<-ch` repeatedly,
**and automatically check `ok` every time, stopping the moment `ok` is
`false`."** That automatic stopping check is the entire reason `range`
exists for channels, not merely a shorter way to write a loop.

**Why plain repeated `<-ch`, with no `ok` check, is actually dangerous:**
remember from the axioms table — once a channel is closed and drained,
`<-ch` never blocks again; it just hands back the zero value, instantly,
forever. A loop written as `for { v := <-ch; ... }`, with no `ok` check, has
no way to notice the channel closed. It would spin forever, "receiving"
endless zero values, never stopping. `range`'s built-in `ok` check is
precisely the difference between "receive some values" and "receive until
told there are no more." (`demo/03-close-range/main.go` shows both styles
side by side on purpose: scene 1 uses `range` and stops silently; scene 2
manually receives with an explicit `ok` check specifically so it can *print*
`ok` at each step and show you the drain-then-`false` transition, which
`range` would otherwise hide by just quietly ending the loop right there.)

**The trap you will absolutely hit** (it's exercise 2): `range ch` exits
*only* on close. If the producer forgets to close, `range` waits forever —
your program hangs with no error message. Symptom from the outside:
"function never returns." First question to ask: *who closes this channel,
and is that line guaranteed to run?*

### The rules of close (memorize — these are interview bait)

1. **Only the sender closes.** Never the receiver. The receiver can't know
   the sender isn't about to send again.
2. **Close exactly once, by exactly one goroutine.** With multiple senders,
   none of them can close (any of the others might still send) — a
   coordinator that *waits for all senders to finish* must do it. The
   standard shape, which combines everything from both modules:

```go
var wg sync.WaitGroup
for i := 0; i < nSenders; i++ {
    wg.Add(1)
    go func() { defer wg.Done(); /* ... send on ch ... */ }()
}
go func() {
    wg.Wait()   // all senders provably finished
    close(ch)   // single closer, after the last send
}()
```

3. **You don't have to close every channel.** Close is only *needed* to end
   a `range`/signal completion. An abandoned channel gets garbage-collected
   fine — unlike files, there's no resource cost to not closing.

### The channel axioms (the classic interview table)

Every operation × every channel state. Fill this table from memory and
you've got the interview question; understand *why* each cell and you've got
the concept:

| Operation | nil channel | open channel | closed channel |
|---|---|---|---|
| **send** `ch <- v` | **blocks forever** | blocks until received / buffer space | **panic!** |
| **receive** `<-ch` | **blocks forever** | blocks until sent / buffer non-empty | zero value, `ok=false`, immediately |
| **close(ch)** | **panic!** | closes it | **panic!** (double close) |

The three panics are all "sender-side contract violations": sending on a
closed channel, closing twice, closing nil. The two "blocks forever" on nil
look useless — until Section 6 shows the `select` trick that makes a nil
channel *deliberately* useful.

▶ **Run:** `go run ./02-channels/demo/03-close-range` — close semantics,
buffer draining, comma-ok, and a recovered send-on-closed panic, all live.

### The generator/producer pattern: send in a goroutine, return the channel

This is **the** standard shape for any function that hands back a channel —
you'll write it constantly from here on, and it's the foundation every
pipeline in Module 4 is built from (it's also, almost verbatim, the
`Generate`/`Squares` example from Rob Pike's original "Go Concurrency
Patterns" talk — this isn't a one-off exercise trick, it's THE idiom):

```go
func Generate(nums ...int) <-chan int {
    ch := make(chan int)
    go func() {
        for _, n := range nums {
            ch <- n
        }
        close(ch)
    }()
    return ch   // returns immediately — does not wait for the goroutine above
}
```

**Why the send has to be inside a goroutine here, specifically:** the
function's whole contract is "return a channel the caller will receive
from." But the caller can't receive from a channel it doesn't have yet — it
only gets `ch` once `Generate` reaches `return ch`. And an unbuffered send
blocks until a receiver exists. So if `Generate` tried to send **before**
returning, that send would block waiting for a receiver — but the only
possible receiver (the caller) can't exist yet, because `Generate` hasn't
returned. Two things each waiting on the other to move first: the same
deadlock shape as trying `ch <- v` then `<-ch` in a single goroutine with no
concurrency at all. The goroutine breaks the cycle: it lets "send the
values" and "return the channel" become two independently-scheduled tasks
instead of one blocked sequence, so `return ch` fires immediately while the
sending continues in the background, waiting patiently for whoever
eventually ranges over the channel.

**Does the goroutine's body definitely start running only *after*
`return ch`?** No — and this is worth being precise about. `go func(){...}()`
is guaranteed to return control to the next line (`return ch`) *without
blocking* — starting a goroutine never waits for it to actually begin
executing. But whether the new goroutine's code has *already started
running* by the time `return ch` executes is not guaranteed either way; on a
multi-core machine the runtime may schedule it onto another thread and start
running it in genuine parallel, possibly reaching `ch <- n` before or after
`return ch` completes — Go makes no promise about the order between them.
**This is harmless, and it's precisely why an unbuffered channel is the
right tool here:** if the goroutine's send reaches the channel first, it
simply **blocks**, patiently waiting for a receiver — it doesn't error, and
it doesn't need `return ch` to have "already happened" in some strict sense.
The channel's own blocking behavior is the synchronization; you never need
to reason about which line technically ran first.

## 5. Channel direction in signatures

A channel variable can be restricted to one direction:

```go
chan int     // bidirectional: can send and receive
chan<- int   // SEND-only  (arrow points INTO the chan: data goes in)
<-chan int   // RECEIVE-only (arrow points OUT of the chan: data comes out)
```

A bidirectional channel converts to either restricted form automatically —
never the reverse. Use this in every function signature you write:

```go
func produce(out chan<- int)      // this function can ONLY send — compiler-enforced
func consume(in <-chan int)       // this function can ONLY receive

func Generate(nums ...int) <-chan int   // caller gets a receive-only view:
                                        // caller CANNOT send on it or close it
```

This is the same discipline as unexported struct fields: the compiler
enforces who's allowed to do what. Returning `<-chan T` from a producer
means no caller can ever violate "only the sender closes" — they physically
can't call `close` on a receive-only channel (compile error). Real codebases
(including shippio's importer) use directional types everywhere; treat them
as mandatory in this module's exercises.

## 6. select: waiting on several channels at once

`select` is `switch` for channel operations: it blocks until **one** of its
cases can proceed, runs that case, and moves on.

```go
select {
case v := <-ch1:
    fmt.Println("ch1 delivered", v)
case v := <-ch2:
    fmt.Println("ch2 delivered", v)
}
```

The rules, each one interview-tested:

- **Blocks until some case is ready.** If several become ready, it picks
  **uniformly at random** — not top-to-bottom. (Deliberate: prevents one
  busy channel from starving the others. Never write code that depends on
  case order.)
- **`default` makes it non-blocking:** if no case is ready *right now*, run
  `default` immediately instead of waiting.

```go
select {
case job := <-queue:
    process(job)
default:
    // queue empty at this instant — do something else, don't wait
}
```

- **Timeout pattern** — the single most-used select idiom in production:

```go
select {
case result := <-resultCh:
    return result, nil
case <-time.After(2 * time.Second):
    return "", fmt.Errorf("timed out")
}
```

**First, what `time.After` actually gives you.** `timeoutCh := time.After(2 *
time.Second)` returns a **channel** immediately — no blocking, no waiting.
That channel starts out empty. Behind the scenes Go starts an internal
timer, and **after exactly that duration elapses, it sends exactly one value
into the channel** (the current time, though you almost never care what the
value actually is). Think of it as a stopwatch-shaped channel that will ding
once, in 2 seconds, no matter what — until then, receiving from it just
blocks like any empty channel.

**Now the whole idiom.** `select` is watching two channels at once, same
rule as always: it blocks until *whichever is ready first* delivers, then
runs that case.

- `resultCh` — some other goroutine is doing real work (a network call, a
  slow query) and will eventually send the actual answer into it.
- `time.After(2 * time.Second)` — a channel that delivers exactly once, in
  2 seconds, guaranteed.

You're racing them. **Whichever delivers first wins** — that's the entire
idiom in one sentence. Real work finishes fast → `resultCh` wins, return the
real result. 2 seconds pass with nothing → the timer wins instead, give up
and return an error.

A full worked example, with a timeline:

```go
func slowCall() string {
    time.Sleep(5 * time.Second) // pretend this is a slow network call
    return "the real answer"
}

func main() {
    resultCh := make(chan string, 1)
    go func() {
        resultCh <- slowCall()
    }()

    select {
    case result := <-resultCh:
        fmt.Println("got:", result)
    case <-time.After(2 * time.Second):
        fmt.Println("gave up waiting")
    }
}
```

```
t=0s   main starts a goroutine to run slowCall()
t=0s   main enters select — now racing two channels
t=0s   time.After's internal timer starts counting down from 2s
t=2s   time.After's channel delivers → select's SECOND case is ready first
       → prints "gave up waiting", select returns
t=5s   slowCall() finally finishes, tries `resultCh <- ...`
       → but nobody's listening anymore (select already moved on)
```

Notice `resultCh` is deliberately **buffered** (`make(chan string, 1)`) —
this is the exact leak issue from Module 1's `ex4`, wearing a channel
costume. At `t=5s`, the slow goroutine still tries to send its late answer.
If `resultCh` were unbuffered, that send would **block forever** — nobody
will ever receive from it again, since `select` already picked the timeout
branch and moved on. The buffer of size 1 gives that late send somewhere to
land instantly, so the goroutine finishes and exits cleanly instead of
leaking. This is precisely what `ex3_first.go` (`FirstResponse`) is asking
you to get right — "the same one idea" as `ex4`, just with a clock added.

**Where you'll actually reach for this:** anywhere you're waiting on
something that might never come back, and refuse to wait forever — calling
an external service ("respond in 3s or I fail the request"), waiting on a
goroutine that might be stuck, or a test that shouldn't hang the whole suite
(exactly what `collectWithTimeout` does in your own `ex1_generate_test.go`:
race the real channel against `time.After` so a hang fails with a clear
message instead of freezing your terminal). One-sentence mental model:
**`select` + `time.After` turns "wait for this" into "wait for this, but not
longer than X" — by racing the real channel against a clock that's also just
a channel.**

(In long-lived loops prefer `time.NewTimer` and stop it explicitly;
pre-Go-1.23 `time.After` held its timer's memory until firing even if you
stopped needing it. For one-shot calls like the exercises above, `time.After`
is fine and idiomatic.)

- **The nil-channel trick:** remember from the axioms that operations on a
  nil channel block forever. Inside a `select`, "blocks forever" means "this
  case can never fire" — so setting a channel variable to `nil` **disables
  that case** without any flags or if-statements:

```go
for ch1 != nil || ch2 != nil {
    select {
    case v, ok := <-ch1:
        if !ok { ch1 = nil; continue }   // ch1 finished: disable its case
        use(v)
    case v, ok := <-ch2:
        if !ok { ch2 = nil; continue }   // ch2 finished: disable its case
        use(v)
    }
}
```

That's the clean way to drain multiple channels to completion — it shows up
again in Module 4's fan-in.

▶ **Run:** `go run ./02-channels/demo/04-select` — multiplexing, random
choice between two ready cases, default, and a timeout, all observable.

## 7. Channels or WaitGroup? (you now own both tools)

The decision framework, which the exercises will burn in:

- **"Wait for N goroutines to all finish, results in known slots"** →
  `WaitGroup` + indexed writes. Simplest, fastest, exactly Module 1.
  A channel adds nothing here.
- **"Consume results AS they're produced"** (streaming, pipeline) →
  channel. A WaitGroup can't do this at all — it has no data path.
- **"First result wins"** → channel (+ buffer to prevent leaks, ex4) or
  channel + `select` timeout. WaitGroup can't express "first."
- **"Hand work from goroutine to goroutine"** (producer/consumer, worker
  pool pulling from a queue) → channel. This is shippio's `queue chan Job`.
- **Both together** is common and correct: channels move the data,
  a WaitGroup tracks "all senders done" so a coordinator can `close` the
  channel (the multi-sender close pattern from Section 4).

Golden rule carried over from Module 1, now with channel teeth: **never
start a goroutine without knowing how it stops.** For channel code the
concrete version is: for every send, know who receives it (or that buffer
space is guaranteed); for every `range`, know who closes.

## 8. Interview questions for this module

1. Walk the channel axioms table from memory: send/receive/close × nil/open/
   closed. Which cells panic, which block forever, and *why* does each make
   sense?
2. When does a send on an unbuffered channel block? Buffered? When does a
   receive block? What's the ONE thing that makes `range ch` terminate?
3. Why must only the sender close a channel? What's the correct close
   pattern with 5 concurrent senders, and which Module 1 tool does it need?
4. `make(chan string, len(queries))` fixed Module 1's `FirstResult` leak —
   explain precisely why the losers no longer leak, and what they do instead.
5. If two `select` cases are ready simultaneously, which runs? Why did Go
   choose that behavior? How do you make a `select` non-blocking?
6. Write the timeout pattern from memory. What does `time.After` return, and
   when does its value arrive?
7. What happens when you receive from a closed channel that still has values
   in its buffer? After it's drained?
8. What is the nil-channel trick in a `select` loop, and what problem does
   it solve?
9. Your function returns `<-chan int`. What two mistakes has the compiler
   just made impossible for your callers?

## 9. This module's files

```
demo/01-unbuffered/     the handshake — sender provably frozen until receiver arrives
demo/02-buffered/       mailbox semantics — watch exactly which send blocks
demo/03-close-range/    close, buffer draining, comma-ok, send-on-closed panic
demo/04-select/         multiplexing, randomness, default, timeout
exercises/              YOUR work — see exercises/README.md
DRILLS.md               predict-the-output drills — answer before running anything
```

```powershell
go run ./02-channels/demo/01-unbuffered     # demos
go test ./02-channels/...                   # fast native runs
.\test-race.ps1 ./02-channels/...           # definition of done
```
