# Module 7 — Interview Gauntlet + Internals

Everything so far taught you to USE Go's concurrency. This module opens
the hood. How is a channel actually built? What really happens when a
goroutine blocks? And when a live system misbehaves, how do you find
the problem with tools instead of guesses?

---

## 0. Why this module exists

Two honest reasons.

**Interviewers probe beneath the API.** Anyone can say "sends block
when the buffer is full." Strong candidates can say WHY: what the
runtime does with the blocked goroutine, where the value goes, who
wakes whom and when. Interviews for senior roles live one level below
the syntax. This module gives you that level.

**Production debugging needs measurement, not vibes.** In Modules 4
and 6 you found goroutine leaks by reading code. That works when the
code is 40 lines. It does not work when the code is 40,000 lines and
the server has been up for nine days. Real engineers capture a profile
and let the runtime tell them where the leak is. This module teaches
those tools: `pprof` and `go tool trace`.

There is also a third half-reason: the gauntlet. Interviews ask you to
BUILD things you have only USED — "implement a rate limiter",
"implement errgroup". Section 5 explains how we train that.

---

## 1. Inside a channel: `hchan`

When you write `make(chan int, 2)`, the runtime allocates one struct
on the heap. It is called `hchan`. Simplified, it holds:

```
hchan {
    buf      circular buffer   // the boxes (dataqsiz slots)
    qcount   int               // how many boxes are full right now
    sendx    int               // where the next send writes
    recvx    int               // where the next receive reads
    recvq    list of waiters   // goroutines parked waiting to RECEIVE
    sendq    list of waiters   // goroutines parked waiting to SEND
    closed   flag
    lock     mutex             // one lock guards all of the above
}
```

That is the whole machine: **one lock, one ring of boxes, two lines of
parked goroutines.** Every channel operation — send, receive, close —
takes that lock first, does a tiny amount of work, and releases it.
A channel is not magic. It is a small, well-guarded data structure.

**The mailbox.** You know it from Module 2 — here is the full picture.
`make(chan int, 3)` is a mailbox with 3 slots. Two kinds of people
visit it: **droppers** (senders) carrying a letter, and **pickers**
(receivers) who want one. Four rules:

1. A picker finds a letter in the box → takes the oldest, leaves.
2. A picker finds the box EMPTY → stands and waits. That waiting line
   IS `recvq`.
3. A dropper finds a free slot → drops the letter in, leaves.
4. A dropper finds the box FULL → stands and waits, letter in hand.
   That line IS `sendq`.

Why can at most ONE of the two lines have people in it? Ask what it
takes to be standing in each. Waiting pickers exist only when the box
is empty (rule 1 — if there were a letter, they'd have taken it and
left). Waiting droppers exist only when the box is full (rule 3 — if
there were a free slot, they'd have used it and left). A box can't be
empty and full at the same time, so the lines can't both be occupied.

### What a send actually does

`ch <- v` takes the lock, then tries three things IN ORDER:

1. **Someone is already waiting to receive?** Pop the first goroutine
   off `recvq` and copy `v` STRAIGHT into that goroutine's stack
   variable. The buffer is never touched. Then mark that goroutine
   runnable. This is called **direct handoff** — the dropper puts the
   letter straight into the waiting picker's hands; it never touches
   the box.
2. **No waiter, but the box has a free slot** (`qcount < dataqsiz`)? Copy
   `v` into the buffer at `sendx`, advance `sendx`, bump `qcount`.
   Done — the sender never blocks.
3. **No waiter and the box is full?** The sender goroutine parks
   itself on `sendq` and tells the scheduler "I can't run until
   someone frees me." This is what "the send blocks" MEANS: the
   goroutine is added to a wait list inside the channel and taken off
   the CPU.

Receive (`<-ch`) is the mirror image: waiting sender? take from them
(and if there's a buffer, take from the buffer head and move the
parked sender's value into the freed slot — the line advances). Buffer
has items? take one. Neither? park on `recvq`.

### The mechanical answers to your old questions

You met all of these behaviors in Modules 2 and 4. Now you can explain
the mechanism, not just the rule.

**Why does receiving from a closed channel never block?**
`close(ch)` sets the `closed` flag and wakes EVERYONE in both wait
queues. From then on, any receive takes the lock, sees `closed == 1`
and an empty buffer, and returns immediately with the zero value and
`ok == false`. There is nothing to wait FOR — the flag answers the
question the receiver was going to wait on. (If the buffer still has
items, receives drain those first; the flag only matters once the
box is empty.)

**Why does sending on a closed channel panic?**
The send takes the lock, sees `closed == 1`, and panics on purpose.
Think about what the alternatives would mean: park forever on `sendq`
(nobody will ever free you — close already emptied the queues), or
silently drop the value (data loss with no error). A loud panic at the
guilty line beats both. This is also WHY the close-ownership rule from
Module 2 exists: only the sender may close, because only the sender
knows no more sends are coming.

**Why does an unbuffered send need a receiver present?**
Unbuffered means `dataqsiz == 0` — a mailbox with ZERO slots. Just a
marked spot on the wall. Nothing can ever be LEFT there, so every
letter must pass hand to hand. Path 2 can never apply: a send either
finds a waiting picker (path 1, direct handoff) or parks (path 3).
That is the whole "synchronization point" behavior from Module 2,
explained by a zero-slot box.

### Numeric walkthrough

`ch := make(chan int, 2)` — a box with 2 slots. Watch the fields:

| Step | Operation | What happens | qcount | sendq | recvq |
|------|-----------|--------------|--------|-------|-------|
| 1 | `ch <- 10` | no waiter, room → buffer[0]=10 | 1 | — | — |
| 2 | `ch <- 20` | no waiter, room → buffer[1]=20 | 2 | — | — |
| 3 | `ch <- 30` | no waiter, NO room → sender parks | 2 | G1 | — |
| 4 | `<-ch` | takes 10 from buffer head, then moves parked G1's 30 into the freed slot, wakes G1 | 2 | — | — |
| 5 | `<-ch` | takes 20 | 1 | — | — |
| 6 | `close(ch)` | closed=1, no waiters to wake | 1 | — | — |
| 7 | `<-ch` | buffer still has 30 → returns 30, ok=true | 0 | — | — |
| 8 | `<-ch` | closed + empty → returns 0, ok=false, NO block | 0 | — | — |

Step 4 is the one people get wrong in interviews: the receive services
BOTH itself and the parked sender, keeping FIFO order (10 came out
before 30). Step 7 and 8 are the "drain then done" behavior of closed
buffered channels.

Run `demo/01-hchan` to watch all of these paths happen live.

---

## 2. The scheduler for real: G, M, P, and work stealing

Module 1 gave you the three letters. Here is the full machine.

- **G** — a goroutine: a small stack (a few KB to start — the
  `02-sched` demo measures ~8 KB each on this machine) plus a program
  counter. A task card.
- **M** — an OS thread. A worker.
- **P** — a "processor": a scheduling seat with its own run queue.
  There are exactly `GOMAXPROCS` of them (default: number of CPU
  cores). A desk.

**The office.** There are only `GOMAXPROCS` desks. A worker (M) must
sit at a desk (P) to run Go code. Each desk has its own tray of task
cards (the **local run queue**, up to 256 cards), and there is one
shared bin in the middle of the office (the **global run queue**).
New goroutines usually land in the tray of the desk that created them.

**Work stealing.** When a desk's tray is empty, the worker does not
sit idle. It checks, in order: its tray → the shared bin → the network
poller (below) → and finally it walks to a RANDOM other desk and
**steals half of that desk's tray**. Half, not one — stealing is a
walk across the office, so make the trip worth it. This is why you
never balance load across goroutines by hand: the scheduler already
rebalances continuously.

### The two kinds of "blocked" — this distinction wins interviews

You have been saying "the goroutine blocks" since Module 2. That word
has been hiding TWO completely different things happening underneath,
and telling them apart is exactly the kind of question that separates
"knows the API" from "knows the runtime."

First, back up to the question underneath: **what does "blocked" even
mean, physically?** Somewhere, some piece of code has been told to
stop running and wait. The question that matters is: **who told it to
stop — the Go runtime, or the operating system?** Those two have
completely different amounts of control, and that difference is the
whole story.

**Case 1 — blocking on a channel, mutex, or WaitGroup.**

Recall the mailbox from Section 1. A picker goroutine finds the box
empty and has to wait. Here is exactly what happens, one step at a
time:

1. The Go runtime (not the OS — this part is entirely Go's own
   bookkeeping) puts that goroutine's task card into the mailbox's
   `recvq` list. That's it — the card just moves from "running" to a
   list sitting inside the channel struct.
2. The runtime tells the desk (the P) this worker is sitting at:
   "that goroutine is done for now — give me another one." This
   function is literally called `gopark` inside the Go runtime source.
3. The **OS thread (the M) never finds out anything happened.** As far
   as the operating system is concerned, that thread is still
   running Go code, at full speed, the entire time. It just switches
   from running goroutine G1 to running goroutine G2, one function
   call apart, in user space, with no trip to the kernel at all.

Say that last part again, because it's the whole point: **the OS does
not know a goroutine "blocked."** The Go runtime handled the entire
thing itself, in userspace, using its own data structures (the wait
queues from Section 1) and its own scheduler code. The thread just
kept working — on a different piece of work.

That is why 100,000 goroutines can sit parked on channels at once and
it costs you almost nothing. Each one is a small stack (Section 2's
top, ~8 KB measured) plus an entry in a linked list. No thread is
tied up. No OS resource is spent. It is bookkeeping, not blocking, in
the OS sense of the word.

**Case 2 — blocking in a syscall (reading a file, a blocking DNS
lookup, calling into C via cgo).**

This is a genuinely different kind of stop. A syscall means the
goroutine has asked the OPERATING SYSTEM to do something — "read the
next 4KB from this file descriptor" — and the OS is the one saying
"wait." The Go runtime has no say in that wait. It cannot un-block a
thread that is sitting inside a kernel read() call; only the kernel
can do that, when the disk hands back the data.

So the OS thread (the M) is now genuinely, physically stuck — parked
by the kernel, not by Go. And that is a problem for the desk (the P)
that thread was sitting at: if the runtime did nothing, that desk
would now sit empty until the disk finishes, even though 15 other
goroutines are ready to run. One slow disk read would freeze an entire
CPU's worth of scheduling capacity.

The runtime's fix, step by step:

1. Just before the thread enters the syscall, the runtime detaches
   the P from that M. The desk and the (about-to-be-stuck) worker
   split up.
2. The runtime hands that now-empty desk to a DIFFERENT worker —
   either a spare thread that was already parked from a previous
   syscall (cheap, no OS involvement), or, if none is spare, the
   runtime asks the OS to create a brand new thread (a real syscall
   itself, `clone` on Linux — more expensive, but rare, since spares
   get reused).
3. That new worker sits down at the freed desk and keeps pulling task
   cards from the tray. Scheduling continues without missing a beat.
4. Meanwhile the ORIGINAL thread is still off in the kernel, blocked,
   carrying exactly one goroutine's worth of work — the one that made
   the syscall. It is not doing anything else and cannot be reused for
   other goroutines while it's stuck there.
5. When the syscall finally returns (the disk delivered the data),
   that goroutine needs a desk again. If a P is free, it grabs one
   immediately. If all P's are already staffed by other workers, the
   goroutine's card goes into a run queue and it waits its turn like
   anyone else. The thread that carried it either becomes the spare
   for next time, or (if there are already enough spares sitting
   around) the runtime lets it exit.

Notice what this means for THREAD COUNT: unlike channel-blocking
(zero new threads, ever), syscall-blocking can genuinely grow the
number of OS threads your program uses — one extra, temporarily, per
goroutine that's stuck in the kernel AT THE SAME TIME. That's the
concrete cost difference.

**Say it back in one line, because this is the sentence an interviewer
wants to hear:** *"Channel blocking is the Go runtime parking a
goroutine in userspace — the thread keeps running other work and no
OS thread is affected. Syscall blocking is the OS itself blocking a
thread — the Go runtime detaches that thread's P and gives the desk to
someone else, so scheduling isn't stalled, but the blocked thread
itself is genuinely stuck until the kernel returns."*

**Network I/O is special** — it does NOT take the syscall path. Waiting
sockets are registered with the OS's event system (epoll on Linux,
IOCP on Windows) in one shared place called the **netpoller**. The G
parks like a channel-block; when the socket becomes ready, the
netpoller hands the G back to a run queue. This is how one Go server
holds 10,000 open connections with a handful of threads.

**Preemption.** Since Go 1.14, a goroutine that runs hot for ~10ms
gets interrupted by the runtime (a background monitor called `sysmon`
spots it, and the G is signaled to yield at the next safe point). So a
tight `for {}` loop can no longer starve everyone else on that P.
`demo/02-sched` shows a heartbeat goroutine surviving next to spinning
loops — before 1.14 that heartbeat would freeze.

### Numeric walkthrough

GOMAXPROCS=2 — exactly two desks, D1 and D2. Two workers, M1 and M2,
start out sitting at them. We will track the OS thread count as we go,
because that number is the entire point of this walkthrough.

**Starting point:** 2 threads total (M1 at D1, M2 at D2). `main`
launches six goroutines, G1 through G6.

**Step 1.** All six new task cards land in D1's tray (the desk that
created them). M1, sitting at D1, starts running G1. D2's tray is
still completely empty — M2 has nothing to do yet.
_Threads so far: 2._

**Step 2.** M2 notices its tray is empty, checks the shared bin (also
empty), then steals. It walks over to D1 and takes HALF of D1's tray —
three cards, say G4, G5, G6. D1 keeps G2 and G3 waiting. M2 starts
running G4.
_Threads so far: still 2 — stealing moves task cards between desks,
it never creates a thread._

**Step 3 — Case 1 happens.** G1 (still running on M1/D1) executes
`<-ch` on a channel with nothing in it. Exactly the mechanism from
the "Case 1" walkthrough above: G1's card goes into that channel's
`recvq`, `gopark` is called, and M1 is told "next card, please." M1
immediately pulls G2 out of D1's tray and starts running it. Notice
what did NOT happen: no thread stopped, no thread was created, no
trip to the kernel occurred. M1 just switched which goroutine it's
executing, in userspace, in the time it takes to call a function.
_Threads so far: still 2._

**Step 4 — Case 2 happens.** G4 (running on M2/D2) calls a function
that reads a file — a genuine syscall. Right before M2 enters the
kernel, the runtime detaches D2 from M2 (step 1 of the Case 2
sequence above). D2 is now an empty desk with a thread stuck in the
kernel formerly sitting at it. The runtime looks for a spare thread to
seat at D2; say one exists from an earlier syscall — call it M3. M3
sits down at D2 and immediately picks up G5 from D2's tray. Meanwhile
M2 is off in the kernel, motionless, waiting on the disk, carrying
only G4.
_Threads so far: 3 — M1, M2 (stuck in the kernel), and M3._

**Step 5.** Some other goroutine sends a value on the channel G1 is
waiting on. Exactly the mailbox mechanic from Section 1: the sender
sees a waiter in `recvq`, hands the value straight to G1, and marks
G1 runnable again. G1's card goes into a run queue (D1's tray, if
there's room, or the global bin). It will resume running the next
time a worker is free to pick it up — not necessarily instantly, but
soon.
_Threads so far: still 3 — this step was pure Case-1 mechanics,
exactly like step 3, so it cost nothing in threads._

**Step 6.** The disk finally returns G4's data. The kernel unblocks
M2. G4 is now ready to keep running. If a desk is free, G4 grabs it
immediately; if not, G4's card waits in a run queue like any other
goroutine until one opens up. M2, no longer carrying anything, becomes
a spare thread — parked, ready to be reused the next time some other
goroutine makes a syscall. The runtime does NOT kill M2 immediately;
keeping a small pool of spares around is cheaper than asking the OS
to create a fresh thread every time.
_Threads so far: still 3 (M2 didn't disappear, it just went idle)._

**The tally at the end:** three OS threads did the work of six
goroutines, with one of those threads only needed TEMPORARILY, for
exactly as long as one file read took. That ratio — a handful of
threads carrying however many thousand goroutines you want — is the
entire reason the G-M-P design exists. Compare it to a language where
every blocking call ties up a full OS thread: six blocking operations
there would cost you six full threads, not three, and definitely not
the two you'd have with zero blocking at all.

---

## 3. pprof: the three profiles that matter

`pprof` is the runtime's built-in census taker. At any moment you can
ask: "what is every goroutine doing right now?", "where are the CPU
cycles going?", "who allocated all this memory?" Each answer is a
**profile** — a set of stack traces with counts attached.

A profile is a **photo**, not a video: aggregate truth at a moment
(or over a sampling window), with no timeline. That's Section 4's job.

### Exposing profiles

Two ways; both are in the standard library.

**For a server** — one import line:

```go
import _ "net/http/pprof"   // registers /debug/pprof/* handlers

go http.ListenAndServe("localhost:6060", nil)
```

**For a program** — write a profile by hand where you need it:

```go
pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)  // runtime/pprof
```

That second form also works INSIDE A FAILING TEST — ex4 uses it to
hand you the leak evidence automatically.

### The goroutine profile — your leak detector

The one you will use most. It lists every live goroutine, GROUPED by
identical stack, with a count per group.

**What "count" actually means here.** The profiler doesn't just dump a
flat list of goroutines. For EVERY live goroutine, it looks at that
goroutine's full call stack — the exact chain of function calls that
got it to wherever it's currently paused — and it GROUPS goroutines
together whenever their stacks are identical. Each group is printed
once, with a number in front: how many goroutines are sitting in that
exact group, right now, at the moment you took the snapshot.

Here's real output, captured from the `03-pprof` demo:

```
goroutine profile: total 117
111 @ 0x7ff7... 0x7ff7... 0x7ff7... 0x7ff7... 0x7ff7...
#	0x7ff7...	main.leakyWorker+0x1b	.../ex4_leakhunt.go:41

1 @ 0x7ff7... 0x7ff7... ...
#	0x7ff7...	runtime/pprof.writeRuntimeProfile+0xb0	...
```

Read the first line as: **111 separate goroutines are all currently
parked at this exact same spot in the code — line 41 of
`main.leakyWorker`.** That `111` is the count. It is a literal
headcount of goroutines frozen at that one line. The second group's
count is `1` — a single goroutine sitting somewhere completely
different (that one's just the profiler capturing itself; ignore it).

Now the leak-hunting recipe, using that number:

1. Capture it: browser to
   `http://localhost:6060/debug/pprof/goroutine?debug=1`, or
   `go tool pprof -top http://localhost:6060/debug/pprof/goroutine`.
2. Read the counts. A healthy server has goroutines constantly coming
   and going — spawned, finished, gone — so every group stays small
   and roughly flat over time. A leak looks different: capture once,
   wait a minute, capture again. Say the `main.leakyWorker` group read
   `111` the first time and `1,311` the second — that one group grew
   by ~1,200 in 60 seconds while every other group stayed flat. THAT
   growing group is the leak, and because pprof already grouped
   identical stacks together, its count comes with the exact file and
   line number attached — no hunting required.
3. Read that group's stack, top frame first. The top frame says WHERE
   they are stuck — `runtime.gopark` → `chansend` means "parked
   sending on a channel", `chanrecv` means "parked receiving",
   `selectgo` means "parked in a select".
4. Read down the stack until you hit YOUR package. That line of your
   code is the send/receive that never completes. Now you know where;
   Module 4 taught you why (nobody left to receive/send) and the fix
   (done channel + select, or context).
5. Fix, redeploy, re-capture. The count must go flat.

`debug=1` gives grouped, human-readable text. `debug=2` gives every
goroutine's full stack separately — noisier, but shows how LONG each
has been blocked (e.g. `chan send, 9 minutes` — nine minutes parked
on a send is a confession).

### The CPU profile — where the cycles go

`go tool pprof -seconds 10 http://localhost:6060/debug/pprof/profile`
samples what is running ~100 times a second for 10 seconds, then drops
you in an interactive prompt. The three commands that matter:

- `top` — functions by CPU time. `flat` = time in the function itself;
  `cum` = time including everything it called.
- `list FuncName` — the function's source with time per line.
- `web` — call graph in the browser (needs graphviz; `top` and `list`
  are enough without it).

### The heap profile — who allocated

`/debug/pprof/heap` — same tooling, answers "which stacks own the
live memory." Deep heap work belongs to Track 2 (escape analysis, GC);
for now you just need to know it exists and how to capture it.

All of this is native on Windows — no Docker, no cgo. Only `-race`
needs the container.

---

## 4. `go tool trace`: when pprof isn't enough

A profile is a photo. **A trace is the security-camera recording.**
It records every scheduling event with timestamps: every goroutine
start, park, unpark, every GC pause, every syscall, laid out per-P on
a timeline you scroll through in the browser.

When you need it: questions about WHEN and IN WHAT ORDER, not "how
much." A pipeline finishes in 4s instead of 1s, but the CPU profile
shows almost nothing — because the time went to WAITING, and waiting
burns no CPU. The trace shows it instantly: stage-2 goroutines sit
parked in neat 300ms stripes, starved by a slow stage upstream.

How to use it:

```go
f, _ := os.Create("trace.out")
trace.Start(f)          // runtime/trace
defer trace.Stop()
```

then:

```
go tool trace trace.out
```

Your browser opens. Two views matter first: **"Goroutine analysis"**
(where each goroutine's time went: running / blocked on channel /
blocked on syscall / waiting for GC) and the **timeline** (per-P
lanes; gaps = idle desks, solid color = work).

You can also label spans of your own code so they show up named on
the timeline:

```go
trace.WithRegion(ctx, "enrich-batch", func() { ... })
```

Cost warning: tracing records everything and is heavy. Capture seconds,
not minutes, and never leave it on in production.

`demo/04-trace` generates a real trace of a deliberately unbalanced
pipeline for you to open and explore.

---

## 5. The gauntlet: rebuilding what you've only used

Everything you have USED as a library, an interviewer can ask you to
BUILD:

- You used `errgroup` in Module 5 → **ex1: build it.**
- You used `rate.Limiter` in Module 6 → **ex2: build it.**
- You used a keyed mutex in Module 6 → **ex3: build its cousin,
  singleflight** (1000 identical requests, one execution, everyone
  gets the answer).

**The gauntlet rule (new for this module):** ex1–ex3 are attempted
COLD, under a 25-minute timer, no hints while the timer runs. When the
timer dies, we drop back to normal hint mode — no shame, the reps are
the point. ex4 (the leak hunt) is untimed; it is a tooling exercise.

**How to narrate while coding** — interviewers grade the talking as
much as the code. Before writing anything, say three things out loud:

1. **The shape.** "Fan-out with a bounded pool" / "one mutex per key" /
   "a struct holding tokens refilled by arithmetic." Name the pattern.
2. **The ownership.** Who closes what; who owns each piece of state.
   You know the rules from Modules 2–4 — SAY them.
3. **The exits.** For every blocking operation: what unblocks it? If
   an answer is "nothing", you have found your own leak before the
   interviewer does.

Then code, and keep muttering the invariants as you go. Silence reads
as confusion even when your code is right.

---

## 6. Interview questions this module answers

1. Walk me through what `ch <- v` actually does, all three paths.
2. Mechanically, why does receiving from a closed channel never block?
   Why does close-then-send panic instead of blocking or dropping?
3. In the buffered-channel walkthrough, a receive freed a slot while a
   sender was parked. Who moves the parked sender's value, and how is
   FIFO order preserved?
4. Draw G-M-P for GOMAXPROCS=2 with six goroutines. Now one blocks on
   a channel and one enters a file-read syscall. What happens to each,
   and how many OS threads are in play?
5. What is work stealing — who steals, from whom, how much, and why
   half?
6. A tight `for {}` loop with no function calls: does it starve the
   other goroutines on its P? Answer for Go 1.13 and for today.
7. Your production server's memory climbs 2% per hour. Walk me through
   finding the cause with pprof — exact steps, exact URLs, what you
   look for in the output.
8. When does a CPU profile mislead you, and a trace tell the truth?
   Give a concrete example.

---

## 7. Files in this module

```
07-internals/
  NOTES.md            ← you are here
  demo/
    01-hchan/main.go      the three send paths + close, live
    02-sched/main.go      preemption + 10k parked goroutines, measured
    03-pprof/main.go      a leaky server to profile for real
    04-trace/main.go      an unbalanced pipeline, traced
  exercises/
    README.md             the gauntlet rules + the table
    ex1_errgrouplite.go   build errgroup (timed)
    ex2_tokenbucket.go    build a rate limiter (timed)
    ex3_singleflight.go   build singleflight (timed)
    ex4_leakhunt.go       find a planted leak WITH pprof (untimed)
  MISTAKES.md
  QUIZ.md
```

Same rules as always: solve until native `go test` passes, then the
real gate — `.\test-race.ps1 ./07-internals/...`.
