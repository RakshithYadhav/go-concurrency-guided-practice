# Module 1 — Goroutines & the Runtime

*Read this top to bottom, running the demos when the text tells you to. Nothing
here assumes you know anything about concurrency yet.*

---

## 0. First, what problem are we even solving?

A normal program is **sequential**: one instruction finishes, the next starts.

```go
resp1 := callServiceA() // takes 100ms
resp2 := callServiceB() // takes 100ms
resp3 := callServiceC() // takes 100ms
// total: 300ms
```

While `callServiceA` waits for the network, your program does... nothing. It
just sits there. B and C don't depend on A's answer, so that waiting is pure
waste. What we *want* is to fire off all three calls, let them wait
*simultaneously*, and collect the answers:

```
sequential:  A ─────► B ─────► C ─────►          total = 300ms
concurrent:  A ─────►
             B ─────►                            total ≈ 100ms
             C ─────►
```

That is **concurrency**: structuring a program as multiple independently
executing tasks. Go's unit for "an independently executing task" is the
**goroutine**, and it's the single biggest reason people pick Go for backend
services.

Two words people mix up constantly — you shouldn't:

- **Concurrency** = *dealing with* many things at once (structure). A single
  barista juggling 5 drink orders is concurrent.
- **Parallelism** = *doing* many things at the same instant (execution). Five
  baristas each making one drink is parallel.

A concurrent Go program *may* also run in parallel if the machine has multiple
CPU cores — but concurrency is about how you structure the code, and that's
what this module teaches.

---

## 1. What is a goroutine?

When your Go program starts, the function `main()` runs inside a goroutine
created for you — the **main goroutine**. So you've been using goroutines all
along; you just only ever had one.

A **goroutine is a function running independently of the code that started
it**, inside the same program, sharing the same memory. You create one by
putting the keyword `go` in front of a function call:

```go
sayHello()      // normal call: main STOPS here until sayHello finishes
go sayHello()   // goroutine: main does NOT stop; sayHello runs "alongside"
```

That one keyword changes the meaning completely:

- **Normal call** — "run this function, wait for it to finish, then continue."
- **`go` call** — "start this function running on its own, and immediately
  continue to my next line. I don't get its return value, and I'm not told
  when it finishes."

Read that last part again, because it's the source of most beginner bugs:
**you don't get the return value, and nobody tells you when it's done.** If
you need a result or need to know it finished, *you* must build that bridge —
with a WaitGroup (this module) or a channel (Module 2).

### A goroutine is not a thread

You may have heard that threads are expensive — that's why languages built on
OS threads (Java pre-virtual-threads, C++) use thread pools carefully. Go's
goroutines are a different animal:

| | OS thread | goroutine |
|---|---|---|
| Starting memory | about 1 MB, fixed | about **2 KB**, grows if needed |
| Cost to create | a system call into the OS — slow | roughly a function call — cheap |
| Who schedules it | the OS kernel | the **Go runtime**, inside your process |
| Sane maximum | a few thousand | **millions** |

This is why Go servers casually start a goroutine *per incoming request*.
`demo/04-scheduler` spawns **100,000 goroutines** in milliseconds on your
laptop — run it later and watch.

---

## 2. The `go` keyword: what runs now vs later

`go f(x)` does two things at two different times, and separating them in your
head will save you real debugging pain:

1. **NOW, in the current goroutine:** the function value `f` and the argument
   `x` are *evaluated*. (What is x's value right now? Which function is f?)
2. **LATER, in a new goroutine:** the call actually executes.

Concretely:

```go
i := 42
go fmt.Println(i) // i is read NOW  -> the goroutine will print 42, guaranteed,
i = 99            // ...even though i changed afterward

go func() {
    fmt.Println(i) // i is read LATER, whenever this goroutine actually runs.
}()               // It might print 99... or something else if i keeps changing.
```

In the first form, `i`'s value is copied into the argument immediately. In the
second, the closure (an anonymous function that *captures* the variable `i`
itself — not its value) reads `i` at some unknown future moment. "Unknown
future moment" + "shared variable" is a recipe for trouble — sections 4 and 5
deal with exactly that.

---

## 3. The first hard rule: `main` does not wait

▶ **Run this now:** `go run ./01-goroutines/demo/01-no-wait` — several times.

```go
func main() {
    for i := 0; i < 3; i++ {
        go func() {
            fmt.Println("worker", i, "reporting in")
        }()
    }
    fmt.Println("main is done")
}
```

Most runs print only `main is done`. Sometimes a worker line sneaks out.
Different runs, different output. Why?

**When `main` returns, the whole process exits. Immediately.** The Go runtime
does not check whether other goroutines are still running. It doesn't ask them
to stop. The operating system tears the process down mid-flight, and any
goroutine that hadn't finished — or hadn't even *started* — simply ceases to
exist.

Here's the timeline of a typical run:

```
main goroutine:      start loop ── go ── go ── go ── print "main is done" ── return ─╳ PROCESS EXITS
worker goroutines:                  (created, waiting for their turn to run...)      ╳ never ran
```

Creating a goroutine only *schedules* it — puts it in a queue of "things to
run". The three workers are sitting in that queue when `main` sprints past
them, prints its message, and returns. Process over.

### Why `time.Sleep` is not the fix

The tempting "fix":

```go
go doWork()
time.Sleep(time.Second) // "surely a second is enough..."
```

This doesn't *synchronize* anything — it just bets that one second is enough.
Some day `doWork` takes 1.1 seconds (slow disk, GC pause, loaded CI machine)
and you lose the bet, silently. Or it takes 5ms and you wasted 995ms every
call. In this repo, `time.Sleep` as a synchronization tool is banned. The real
fix is a tool that makes main wait *exactly* until the work is done — that
tool is next.

---

## 4. `sync.WaitGroup`: "wait until my N goroutines finish"

A `WaitGroup` is nothing magical — it's a **thread-safe counter with a wait
button**:

| Method | What it does |
|---|---|
| `wg.Add(n)` | counter += n — "n more tasks are in flight" |
| `wg.Done()` | counter -= 1 — "one task just finished" |
| `wg.Wait()` | **block right here** until the counter reaches 0 |

The pattern, which you'll write hundreds of times:

```go
var wg sync.WaitGroup

for _, url := range urls {
    wg.Add(1)               // (1) BEFORE go: "one more in flight"
    go func() {
        defer wg.Done()     // (2) guarantees the decrement, even on panic
        check(url)          // (3) the actual work
    }()
}

wg.Wait()                   // (4) blocks until every Done has fired
// safe: all work is finished past this line
```

Walking through it with 3 URLs:

```
main:    Add(1) Add(1) Add(1)   ...counter = 3...   Wait() ──── blocked ────► unblocked, continue
worker1:        check(url) ─► Done()   counter = 2
worker2:          check(url) ───► Done()   counter = 1
worker3:       check(url) ──────────► Done()   counter = 0  ← Wait releases
```

▶ **Run:** `go run ./01-goroutines/demo/02-waitgroup` — all workers always
print (completion is guaranteed) but in a different order each run (ordering
is not). Both halves of that sentence matter.

### The rules, and *why* each one exists

**Rule 1 — `Add` goes BEFORE `go`, in the parent goroutine. Never inside the
child.** This is the classic interview trap, and exercise 3 makes you fix it.
Here's the buggy version and the exact failure sequence:

```go
for _, url := range urls {
    go func() {
        wg.Add(1)        // BUG: runs whenever the goroutine gets scheduled
        defer wg.Done()
        check(url)
    }()
}
wg.Wait()
```

Failure, step by step:

1. `main` launches 3 goroutines. Launching just *queues* them — none has
   necessarily run a single instruction yet.
2. `main` reaches `wg.Wait()`. The counter is still **0**, because every
   `Add(1)` lives inside goroutines that haven't started.
3. `Wait()` sees counter == 0 and — correctly! — returns immediately.
4. `main` continues, maybe returns, process exits... while workers are only
   now getting around to `Add(1)`.

`Wait` didn't malfunction. It was asked "is the counter zero?" and it was. The
bug is that the bookkeeping ("one more in flight") happened *after* the race
had already been lost. Put `Add` in the parent, before `go`, and the counter
is provably correct by the time `Wait` runs.

**Rule 2 — `defer wg.Done()`, first line of the goroutine.** `defer` runs even
if the function panics or returns early. A forgotten/skipped `Done` means the
counter never hits 0 and `Wait()` blocks *forever* — a deadlocked program.

**Rule 3 — don't copy a WaitGroup.** It's a counter; a copy is a *different
counter*. If a function needs one, pass a pointer (`*sync.WaitGroup`).
`go vet` catches copies — one more reason vet runs on everything here.

### The modern shorthand: `wg.Go`

Go 1.25 added a method that bundles the whole ceremony:

```go
var wg sync.WaitGroup
for _, url := range urls {
    wg.Go(func() { check(url) })   // = Add(1) before + go + deferred Done
}
wg.Wait()
```

Use it in new code if you like — but you MUST be fluent in the manual form:
it's in every codebase written before 2025 and in every interview.

---

## 5. Closures and the loop-variable story

### What a closure captures

```go
count := 0
increment := func() { count++ }  // closure: captures the VARIABLE count
increment()
increment()
fmt.Println(count) // 2
```

The closure doesn't get a copy of `count` — it holds a reference to *the
variable itself*. When a closure runs in a goroutine, that means two
goroutines can be looking at the same variable. Keep that loaded in your head.

### The famous gotcha (pre-Go 1.22)

For a decade, this was the most common Go bug in existence:

```go
for i := 0; i < 3; i++ {
    go func() { fmt.Println(i) }()
}
// pre-Go 1.22: usually printed "3 3 3"
```

Why: before Go 1.22 there was **one** variable `i` for the *entire loop*,
updated each iteration. All three closures captured that same single variable.
By the time the goroutines actually ran, the loop had finished and `i` was 3.
All three printed the final value.

The two classic fixes (recognize both on sight):

```go
go func(i int) { fmt.Println(i) }(i)  // fix 1: pass as argument — evaluated NOW, copied

for i := 0; i < 3; i++ {
    i := i                            // fix 2: shadow — fresh variable per iteration
    go func() { fmt.Println(i) }()
}
```

### What changed in Go 1.22 (you're on 1.26)

The language was changed: **each loop iteration now gets its own fresh
variable**. The snippet above now correctly prints 0, 1, 2 (in some order).
The trap, as written, is dead.

So why still learn it?

1. Interviews ask it constantly ("what does this print and why?"). The full
   answer: *"Pre-1.22: one shared loop variable, closures see the final value;
   fixed by parameter-passing or shadowing. Go 1.22 made iteration variables
   per-iteration, so it now prints 0,1,2."* That answer signals you actually
   understand capture.
2. The underlying rule is very much alive: closures capture **variables, not
   values**. The loop variable got special treatment in 1.22 — but a variable
   declared *outside* the loop, a struct field, a slice you're writing into —
   share any of those between goroutines and you're in race territory. Which
   is exactly section 6.

---

## 6. Data races — the worst bug class in concurrent code

### Definition

A **data race** happens when:

1. two goroutines access the **same memory** (same variable, same map, same
   slice index),
2. **at least one access is a write**, and
3. **nothing synchronizes them** (no mutex, no channel, no WaitGroup edge —
   nothing establishing who goes first).

### Why `count++` from two goroutines loses updates

`count++` looks atomic. It isn't. The machine executes three separate steps:

```
1. LOAD  count from memory into a register
2. ADD   1 to the register
3. STORE the register back to memory
```

Now run two goroutines, both incrementing, and watch one interleaving:

| step | Goroutine A | Goroutine B | memory value of count |
|------|-------------|-------------|----------------------|
| 1 | LOAD → sees 5 | | 5 |
| 2 | | LOAD → sees 5 | 5 |
| 3 | ADD → register 6 | | 5 |
| 4 | | ADD → register 6 | 5 |
| 5 | STORE 6 | | 6 |
| 6 | | STORE 6 | **6** ← should be 7! |

Two increments happened; the count went up by one. B never saw A's increment
because both LOADed before either STOREd. Multiply by thousands of increments
and you get the demo below losing tens of thousands of updates.

▶ **Run both of these and compare:**

```powershell
go run ./01-goroutines/demo/03-race        # wrong total. NO error. Just... wrong.
.\test-race.ps1 -run TestWordFrequency ./01-goroutines/exercises   # race detector screaming
```

The scary part of the first run: **no crash, no error message — just a wrong
number.** In production that's a corrupted invoice total, a wrong account
balance. And a racy program isn't merely "sometimes off": once a data race
exists, the Go memory model makes **no guarantees at all** about what the
program does. Compilers and CPUs reorder operations assuming no races; break
the assumption and "impossible" things happen. There is no such thing as a
benign data race.

### The race detector

Go ships a real weapon for this:

```bash
go test -race ./...     # (on this machine: .\test-race.ps1 ./...)
go run  -race main.go
```

`-race` rebuilds your program with every memory access instrumented. When two
unsynchronized accesses touch the same address, it prints **both stack
traces** — the "who read" and the "who wrote". You saw that output when you
ran the exercise 2 test above.

Two things to internalize about it:

- It only reports races that **actually occurred during that run**. It never
  false-alarms (a report = a real race), but a clean run doesn't *prove* the
  code race-free — maybe the racy interleaving just didn't happen this time.
  That's why the tests here hammer functions hundreds of times: more
  interleavings, better odds of exposing the race.
- It costs roughly 2–10x speed and more memory. Nobody cares in tests. Always
  run tests with it. (One wrinkle: on Windows `-race` needs a C toolchain, so
  in this repo `.\test-race.ps1` runs the tests inside a Linux Docker
  container instead. Same detector, same output.)

---

## 7. Under the hood: how Go runs a million goroutines

*You can do this whole module treating the scheduler as a black box. But "how
does Go actually schedule goroutines?" is a favorite senior-level interview
question, and the mental model explains a lot of observable behavior. Here's a
beginner-friendly version.*

The problem: the OS gives you *threads* (expensive, ~1MB each, slow to switch)
and you want *goroutines* (cheap, millions). So the Go runtime builds its own
scheduler that maps many goroutines onto few threads — called **M:N
scheduling**. It has three moving parts, named G, M and P:

**A restaurant kitchen analogy:**
- **G (goroutine)** = an order ticket — a dish someone asked for, plus where
  you got up to in cooking it.
- **M (machine = OS thread)** = a cook. Cooks do the actual work.
- **P (processor = scheduling context)** = a kitchen *station* with a rail of
  pending order tickets (its **local run queue**). A cook must stand at a
  station to cook.

The number of stations is **GOMAXPROCS** — by default, the number of CPU
cores. That number is the answer to "how many goroutines can be executing Go
code *at the same instant*?" You can have a million tickets (goroutines), but
with 8 stations only 8 dishes are actively being cooked at any moment;
everything else is queued.

How work flows:

- Each P (station) runs goroutines from its own local queue, one after
  another. There's also one **global queue** for overflow.
- A P whose queue is empty doesn't idle — it **steals half the tickets** from
  another P's rail. This is *work stealing*, and it keeps all cores busy
  without any central coordinator.

### What happens when a goroutine blocks — the part that makes Go special

This is why Go I/O code can be written in a simple blocking style and still
scale:

- **Goroutine blocks on a channel or mutex** (pure Go-level waiting): the
  runtime just *parks* the G — sets the ticket aside — and the M instantly
  picks the next G from the queue. Cost: nanoseconds. No OS involvement. This
  is why demo 04 can park 100,000 goroutines without breaking a sweat.
- **Goroutine makes a blocking system call** (e.g. reading a file): the OS
  thread (M) itself is now stuck — the cook is trapped waiting at the oven.
  The runtime's move: **detach the station (P) from the stuck cook and hand it
  to another cook (M)**, so the station keeps processing other tickets. When
  the syscall finishes, the old cook rejoins. Result: one slow syscall never
  stalls the other goroutines.
- **Network I/O**: even better — Go routes it through the **netpoller**
  (epoll/kqueue/IOCP under the hood), so a goroutine waiting on a socket
  doesn't hold *any* thread. 100,000 idle connections ≈ 100,000 parked Gs ≈ a
  few hundred MB of small stacks, a handful of threads. This is the
  architectural reason Go eats high-connection servers for breakfast.

### Preemption — can one goroutine hog a station forever?

It used to be possible: scheduling decisions happened at function calls, so a
tight `for {}` loop with no calls could starve everyone on that P. Since **Go
1.14** the runtime does **asynchronous preemption**: after ~10ms it signals
the hog and reschedules it. A busy loop still burns a core, but it can no
longer freeze the scheduler.

Odds and ends you'll actually use:

- `runtime.NumGoroutine()` — how many goroutines exist right now. The leak
  test in exercise 4 uses it: goroutine count that grows and never comes down
  = leak.
- `runtime.GOMAXPROCS(0)` — read the current setting without changing it.
- Since Go 1.25, GOMAXPROCS respects container CPU limits — your program in a
  "2 CPUs" Kubernetes pod defaults to 2, not the host's 64.

▶ **Run:** `go run ./01-goroutines/demo/04-scheduler` — then re-run with
`$env:GOMAXPROCS='1'` and watch the CPU-bound phase slow down by ~Ncores while
everything else barely changes. Concurrency without parallelism.

---

## 8. When should you actually reach for `go`?

Not "whenever things should be fast". The three legitimate triggers:

1. **I/O fan-out** — N independent network/disk operations; run them
   concurrently and total time ≈ the slowest one instead of the sum.
   (Exercise 1 is exactly this.)
2. **Background work** — the caller doesn't need the result *now*. Your
   shippio importer is the canonical example: the HTTP handler returns
   `202 Accepted` immediately and worker goroutines do the import later.
3. **CPU parallelism** — genuinely CPU-bound work split across cores. Only
   pays off if the work is big enough to dwarf the coordination overhead.

And the golden rule, worth memorizing verbatim:

> **Never start a goroutine without knowing how it will stop.**

Before every `go`, answer: who waits for this goroutine? What makes it exit?
What happens to it if the rest of the program moves on? Exercise 4 shows the
alternative — goroutines that can *never* exit, quietly stacking up until the
process dies. In a long-running server, that's a slow-motion outage.

---

## 9. Interview questions

At review time you answer these out loud, no notes. The narration practice
matters as much as the code.

1. What happens to running goroutines when `main` returns? Why is `time.Sleep`
   never an acceptable way to wait for them?
2. Why must `wg.Add(1)` be called before `go func(...)`, not inside the
   goroutine? Narrate the exact failure sequence (there are 4 steps).
3. Explain the pre-1.22 loop-variable capture bug, its two classic fixes, and
   what Go 1.22 changed. What's the general closure-capture rule that is
   *still* true?
4. What three conditions define a data race? Why can `x++` from two goroutines
   lose updates — walk the LOAD/ADD/STORE interleaving. What does a clean
   `-race` run prove, and what does it not prove?
5. Explain G, M, and P in one sentence each. What does the runtime do with the
   P when its M enters a blocking syscall, and why is network I/O cheaper
   still?
6. A new goroutine's stack is roughly what size, and why does that make
   goroutine-per-request a sane server design?
7. State the golden rule of starting goroutines, and use exercise 4's
   `FirstResult` to illustrate breaking it.

---

## 10. Your files

```
demo/01-no-wait/        main returns before goroutines run — run it first
demo/02-waitgroup/      the fix, classic form + wg.Go shorthand
demo/03-race/           deliberate data race — run plain, then under -race
demo/04-scheduler/      100k goroutines, GOMAXPROCS, parked-vs-running
exercises/              YOUR work — see exercises/README.md
```

```powershell
go run ./01-goroutines/demo/01-no-wait      # demos: native go run
go test ./01-goroutines/...                 # fast native tests (no race detection)
.\test-race.ps1 ./01-goroutines/...         # the real gate: -race via Docker
```
