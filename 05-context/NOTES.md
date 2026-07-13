# Module 5 — context & Graceful Shutdown

## 0. Why this module exists

In Module 4 you built the done-channel pattern: an extra channel whose
only job is to tell goroutines "stop, the consumer left." It works. Now
look at what it can't do:

- It has no **deadline**. "Stop after 2 seconds" means hand-wiring a
  timer next to every done channel, every time.
- It has no **reason**. When work stops, nobody can ask "was that a
  timeout? a canceled request? a shutdown?" — the channel is just closed.
- It doesn't **compose**. A real request passes through your handler,
  then your service layer, then the database client, then an HTTP call
  to another API. A hand-made done channel would have to be threaded
  through every one of those layers by hand, and none of the libraries
  you call know about YOUR channel.

Go's answer is `context.Context` — the done channel, standardized, with
deadlines and reasons built in, and adopted by essentially every library
that does I/O. `database/sql`, `net/http`, gRPC, Redis clients, cloud
SDKs: they all take a `ctx` as their first argument. This is not an
optional style choice in production Go. **Fluency with context is the
difference between code that can be stopped and code that can only be
killed.**

And "being stoppable" has a name in production: **graceful shutdown** —
the sequence your shippio server runs when Kubernetes sends it SIGTERM.
That's the second half of this module, and it's where every pattern from
Modules 1–4 finally clicks together into one production shape.

### The story that makes the need obvious

A user hits your search endpoint. Your handler fans out: one Postgres
query (heavy — ~10 seconds on a bad day), one call to a recommendations
service, one cache lookup. Half a second in, the user closes the tab.

Now ask: **who tells the database to stop?**

Without context: nobody. The connection to the user is gone, but your
handler doesn't know that, so it keeps waiting. Postgres keeps grinding
through the heavy query — 10 more seconds of CPU and disk spent on an
answer that will be thrown away the moment it's ready. Multiply by a
traffic spike: hundreds of doomed queries piling up on your database,
slowing down the queries that still matter, for users who already left.
This is a real production failure mode.

Work that outlives its usefulness doesn't just waste time. It holds
memory, connections from the pool, file handles, goroutines. A server
that can't cancel doomed work falls over during exactly the traffic
spikes it most needs to survive.

### Why your done channel can't solve this

You could wire a done channel through YOUR own code. But follow the call
chain of that search request:

```
your handler → your service layer → database/sql → the driver → the socket
```

Your hand-made done channel dies at the third arrow. `database/sql` is
not your code — it has never heard of your channel and has no parameter
to accept it. For a stop signal to travel the WHOLE chain, from "user
closed the tab" all the way down to "kill the query on the wire," every
layer — including code written by strangers — has to agree on one
standard way to receive it.

That is the real answer to "why does every library take a ctx": the
value of context is not the mechanism. You built the mechanism yourself
in Module 4 ex3 — it's just a done channel. The value is the
**agreement**. One interface, always the first parameter, that every
library accepts — so cancellation composes across code written by people
who never met. `net/http` cancels the request's context the moment the
client disconnects; that context flows into your handler, through your
service code, into `db.QueryContext`, and the driver kills the actual
query on the wire. You wrote none of that plumbing. It works because
everyone speaks the same word.

The second reason is **time budgets**. Real systems need "you have this
much time" to travel down the chain too. Your service promises answers
in 2 seconds; it calls service B, which calls service C. If C is having
a bad day, everyone above it must give up on schedule, not pile up
waiting. A context carries a deadline down through every layer
automatically — each layer can even ask "how much budget is left?" and
skip work that can't finish in time.

**The one-sentence version:** in real systems, the answer to "should
this work keep going?" can change while the work is running — and the
only way for that news to reach every layer, including other people's
code, is one universal, agreed-upon signal. Everything else about
context — the tree, `Done()`, `Err()`, deadlines — is engineering
around that agreement.

## 1. What a context actually is

Strip away the fear: a `Context` is a small object carrying three things.

1. **A done channel.** `ctx.Done()` returns `<-chan struct{}` — literally
   the same kind of channel you built in Module 4. It closes when the
   context is canceled. You select on it exactly the same way.
2. **A deadline (maybe).** "This work must stop by 14:03:05.000."
3. **A reason.** After cancellation, `ctx.Err()` tells you which kind of
   stop it was.

That's it. Everything else is plumbing for one big idea:

### Contexts form a tree, and cancellation flows DOWN it

You never modify a context. You **derive** a child from a parent:

```go
ctx := context.Background()                        // the root: never canceled
reqCtx, cancel := context.WithCancel(ctx)          // child of root
dbCtx, cancel2 := context.WithTimeout(reqCtx, 2*time.Second)  // child of child
```

Think of a chain of command. The general (root) gives an order to a
colonel (request context), who gives it to a sergeant (database call
context). If the colonel calls off the operation, the sergeant and
everyone below stop too — **canceling a parent cancels every descendant**.
But it never flows up: the sergeant hitting his 2-second limit stops only
his own little operation. The colonel and the general don't even notice.

Why is this tree shape the right design? Because it matches how real
requests work. One HTTP request fans out into three database queries and
two API calls. If the user closes the browser tab, ALL of that downstream
work is now pointless — one cancel at the request level should sweep it
all away. But one slow database query timing out shouldn't kill the whole
request if the handler wants to try a fallback. Down, not up.

▶ **Run:** `go run ./05-context/demo/01-tree` — a three-level tree,
canceled in the middle. Watch who stops and who keeps running.

## 2. The API you'll actually use

### Roots

```go
context.Background()   // the standard root — main(), tests, servers
context.TODO()         // "I haven't wired context through here yet" — a marker
```

Both are empty, never-canceled roots. `TODO` exists purely so you can
grep for unfinished plumbing later.

### Deriving children

```go
ctx, cancel := context.WithCancel(parent)             // cancel by hand
ctx, cancel := context.WithTimeout(parent, 2*time.Second)   // auto-cancel after duration
ctx, cancel := context.WithDeadline(parent, tenPM)    // auto-cancel at a wall-clock time
```

(`WithTimeout` is just `WithDeadline(parent, time.Now().Add(d))` — same
machinery, two spellings.)

### The rule that MUST become reflex: `defer cancel()`

Every derive gives you a `cancel` function, and you must call it —
even when the work finishes fine, even on a timeout context that "will
cancel itself anyway":

```go
ctx, cancel := context.WithTimeout(parent, 2*time.Second)
defer cancel()
```

Why? When you derive a child, the parent records it (that's how
cancellation flows down the tree), and a timeout context also starts a
timer. `cancel()` is what unhooks the child from the parent and stops
the timer. Skip it and those stay alive until the timeout fires or the
parent is canceled — which for a long-lived server's Background parent
is "essentially forever." It's a context leak: the same disease as
Module 4's goroutine leak, wearing different clothes. `go vet` flags
missing cancels for a reason. `defer cancel()` on the next line, every
time, no exceptions.

### Asking a context what happened

```go
ctx.Err()      // nil while alive; context.Canceled or context.DeadlineExceeded after
ctx.Done()     // the channel — select on it
```

Two stop reasons, and production code branches on them: `Canceled` means
someone deliberately called cancel ("user closed the tab"), while
`DeadlineExceeded` means time ran out ("the database is slow"). One is
usually fine to log quietly; the other is usually an alert.

### The modern additions (Go 1.20–1.21) — know they exist

- **`context.WithCancelCause` + `context.Cause(ctx)`** — like WithCancel,
  but you cancel WITH an error: `cancel(errors.New("user quota hit"))`.
  `ctx.Err()` still says just `Canceled`; `Cause(ctx)` returns your real
  reason. Fixes the oldest context complaint: "it stopped, but WHY?"
- **`context.WithoutCancel(parent)`** — a child that keeps the parent's
  values but IGNORES its cancellation. The real use: a request is
  canceled, but you still need to write the audit log — that write must
  not die with the request.
- **`context.AfterFunc(ctx, f)`** — run `f` in its own goroutine when ctx
  is canceled. Cleanup hooks without hand-writing the watcher goroutine.

Survey-level for now — recognize them, reach for them when the need
appears.

### Values: the feature you should mostly NOT use

`context.WithValue(ctx, key, val)` stashes request-scoped data in the
context. The rule: **only data about the request that middleware needs
and that no function signature should carry** — trace IDs, request IDs,
the authenticated user. Never function parameters in disguise. If a
function needs a limit or a filename, put it in the signature where the
compiler can see it. Hiding real parameters in a context turns compile
errors into runtime surprises.

## 3. The rules of ctx in APIs (idiom, memorize)

1. **First parameter, named `ctx`.**
   `func Fetch(ctx context.Context, url string) error` — always first,
   by convention so strong that linters enforce it.
2. **Never store a context in a struct.** A context belongs to ONE call's
   lifetime. Structs outlive calls. (The rare exceptions know they're
   exceptions.) Pass it down the call chain instead.
3. **Never pass nil.** Don't have a real one? `context.TODO()`.
4. **Don't cancel what you didn't create.** Whoever derives the context
   owns the cancel. Callees just listen.

## 4. Making YOUR code actually respect a context

Taking `ctx` as a parameter is decoration until your code actually
listens to it. Three places to listen:

### Channel operations: select, exactly like Module 4

```go
select {
case out <- v:          // normal path
case <-ctx.Done():      // canceled — stop cleanly
    return ctx.Err()
}
```

This IS Module 4's done-channel pattern. `ctx.Done()` just replaces your
hand-made `done`. Every send and every receive in cancellable code gets
this treatment.

### Long loops: check as you go

A CPU-heavy loop with no channel operations never touches `select`, so
it would never notice cancellation. Give it a pulse check:

```go
for i, item := range hugeList {
    if err := ctx.Err(); err != nil {
        return err                      // someone said stop — obey promptly
    }
    process(item)
}
```

### I/O: pass ctx to the library — it does the hard part

```go
req, err := http.NewRequestWithContext(ctx, "GET", url, nil)   // HTTP
rows, err := db.QueryContext(ctx, "SELECT ...")                // SQL
```

This is the payoff of the whole convention: the library aborts the
in-flight network call the moment your ctx cancels. Your hand-made done
channel could never reach inside `database/sql`. Context can, because
everyone agreed on the same interface.

## 5. errgroup — WaitGroup, evolved for the real world

### The problem, restated with real stakes

Recall Module 1's `FetchAll`: a WaitGroup plus slots, N goroutines all
doing their own independent work, main waiting for every one of them to
finish. That shape has a real gap once failure enters the picture.

Picture this: you send five friends to five different stores to check if
a sold-out item is back in stock. Ten minutes in, friend #3 calls: "I
checked — the manufacturer discontinued this item completely. Nobody
will ever have it." What should happen next?

With plain `WaitGroup`, the honest answer is: nothing happens. The other
four friends have no way of knowing. They keep driving, keep checking,
keep burning gas and time, for a search that is now provably pointless.
You only find out it was all wasted once the LAST friend finally calls
back, whenever that happens to be.

That's the gap. Two things are missing: a way to **collect the failure**
the moment it happens, and a way to **tell everyone else to stop**,
instead of letting them run a doomed errand to completion. Hand-wiring
both yourself, every time, means a WaitGroup, plus an error variable
guarded by a mutex (or a channel), plus your own cancel signal threaded
through every goroutine. `errgroup` (package `golang.org/x/sync/errgroup`)
packages all of that into a few method calls.

### The code, walked through line by line

```go
g, ctx := errgroup.WithContext(parentCtx)

for _, url := range urls {
    g.Go(func() error {
        return fetch(ctx, url)      // note: uses the GROUP's ctx
    })
}

if err := g.Wait(); err != nil {    // waits for all; returns FIRST error
    return err
}
```

**`g, ctx := errgroup.WithContext(parentCtx)`** — this single line does
two things at once, and both matter. It builds `g`, the group itself
(the thing tracking all your goroutines, like a WaitGroup does). And it
derives a BRAND NEW context — call it the group's own ctx — as a child
of `parentCtx`. This is not the same object as `parentCtx`. It's a fresh
child, specifically created so `errgroup` can cancel it later, on its
own, without touching `parentCtx` at all. Hold onto that distinction —
it's the crux of the whole mechanism, and it's exactly why the next line
matters so much.

**`g.Go(func() error { return fetch(ctx, url) })`** — starts a goroutine,
the same way `wg.Add(1); go func() { defer wg.Done(); ... }()` did in
Module 1, except `errgroup` handles the `Add`/`Done` bookkeeping for you.
Notice which `ctx` gets passed into `fetch`: the GROUP's ctx from the
line above — NOT `parentCtx`. This is not a style choice. If you passed
`parentCtx` here instead, none of these fetches would ever learn about
each other's failures, because `parentCtx` is never the thing `errgroup`
cancels. Only the group's own derived ctx gets canceled on failure — so
that is the only ctx your workers can listen to if you want them to hear
about it.

**`g.Wait()`** — blocks until every single goroutine started with `g.Go`
has returned, exactly like `wg.Wait()`. The difference: it also gives you
back a return value — the FIRST non-nil error to come out of any of them
(later errors, if there are several, are simply dropped, on purpose,
because the first failure is almost always the real root cause; the rest
are usually just the fallout).

### The actual cancellation trick — the part that solves the friends-and-stores problem

Here's the mechanism, stated directly: **the moment ANY of your `g.Go`
functions returns a non-nil error, `errgroup` immediately cancels the
group's own ctx** — the one you got back from `WithContext`, the one you
were told to pass into every worker. That cancellation is `errgroup`
doing exactly what you'd want a good group leader to do: the instant one
search comes back "this is pointless," it radios everyone else to turn
around.

But — and this connects straight back to the closing-bell idea from
Section 6 — `errgroup` **cannot force any goroutine to stop.** It can
only cancel that shared ctx. Whether a given `fetch` call actually
notices and turns around depends entirely on whether IT is written to
listen — a `select` against `ctx.Done()`, same as always. An `errgroup`
worker that ignores its ctx will just keep running to completion,
wasting time, exactly like the friend who never checks their phone.
Passing the right ctx gets you the ABILITY to stop early. The worker
function still has to actually listen for that to happen.

### One more piece: bounding how many run at once

`g.SetLimit(n)` caps how many of your `g.Go` goroutines run at the same
moment — Module 4's bounded parallelism (the semaphore pattern), now
built into the group with one method call instead of a hand-rolled
buffered channel.

### The mental model to keep

**WaitGroup is for goroutines that can't fail.** `errgroup` is for
goroutines that can — which, in production, is almost always true. Any
time you're about to reach for a `WaitGroup` and also find yourself
wanting "collect the error" or "stop the others if one fails," that's
the signal to reach for `errgroup` instead.

▶ **Run:** `go run ./05-context/demo/03-errgroup` — five fetches, one
fails at 50ms, watch the other four get canceled instead of running
their full second.

## 6. Graceful shutdown — the production finale

### What "graceful" actually means

Your server is mid-flight: 30 requests being handled, 12 jobs in the
worker pool. Kubernetes sends SIGTERM (the polite "please exit" signal —
`kill <pid>`, deploys, autoscaling). Two ways to respond:

- **Ungraceful:** process exits now. 30 users get connection-reset
  errors. 12 jobs die half-done — and if they were payments, you now
  don't know which half happened.
- **Graceful:** stop taking NEW work, finish (drain) the work in hand,
  then exit. With a time limit, because "graceful" can't mean "forever" —
  Kubernetes only waits ~30s before SIGKILL, which cannot be caught.

### The four-step shape (memorize this order)

```
1. HEAR the signal        (SIGTERM/SIGINT → a canceled context)
2. STOP the intake        (close the listener / stop accepting jobs)
3. DRAIN in-flight work   (with a deadline)
4. EXIT                   (cleanly if drained, forcefully at the deadline)
```

### Step 1 is one line now

```go
ctx, stop := signal.NotifyContext(context.Background(),
    os.Interrupt, syscall.SIGTERM)
defer stop()
```

`signal.NotifyContext` gives you a context that cancels when the OS
signal arrives. The entire operating-system side of shutdown collapses
into "a context canceled" — which your whole program already knows how
to respond to, because everything downstream listens to ctx. One
mechanism, top to bottom.

### Steps 2–4 for an HTTP server

```go
srv := &http.Server{Addr: ":8080", Handler: mux}

go func() { srv.ListenAndServe() }()   // serve in background

<-ctx.Done()                           // block here until the signal

shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
err := srv.Shutdown(shCtx)             // stops intake + drains, all in one
```

`srv.Shutdown` does steps 2 and 3 for you: closes the listener (new
connections refused), then waits for in-flight requests to finish — up
to `shCtx`'s deadline, after which it gives up and returns an error.

One subtlety worth staring at: the drain context is derived from
`context.Background()`, NOT from the signal ctx. Why? The signal ctx is
**already canceled** — that's why we're shutting down. A child of it
would be born dead, and the drain would abort instantly instead of
giving requests their 10 seconds. The shutdown clock must be a fresh
timer, independent of the thing that triggered it.

### Three questions worth being able to answer about this exact snippet

**Why doesn't cancellation forcibly stop anything?** Picture context as a
closing bell, not a security guard who checks on people. `cancel()` (or
a timeout firing) does exactly ONE thing: it closes `ctx.Done()`, once.
Nobody walks around checking on goroutines or dragging them out — Go has
no way to force a goroutine to stop from the outside. It's on EACH
goroutine to actively check whether the bell has rung (`select` on
`<-ctx.Done()`, or a periodic `ctx.Err()` check) and choose to leave.
A goroutine that never checks — like `ThumbnailAll` before its fix —
just keeps working, completely oblivious, long after the bell rang.
Cancellation is a signal to notice, not a switch that flips anything off
by itself.

**Does `<-ctx.Done()` blocking main freeze the server too?** No — and
this is worth tracing carefully, because it looks alarming at first.
`go func() { srv.ListenAndServe() }()` and `<-ctx.Done()` are two
independent goroutines. Blocking one goroutine never freezes another —
that's the same "main doesn't wait" idea from Module 1, just flipped:
here main deliberately waits while a SEPARATE goroutine (the server)
keeps working the whole time, completely unaffected. Main has nothing
useful left to do at this point except wait for the shutdown signal, so
it blocks on purpose, cheaply, doing nothing until the signal arrives.

**Why does `ListenAndServe` need the `go func()` wrapper at all?**
Because `ListenAndServe` never returns on its own — it serves requests
forever until something stops it. Without `go func()`, main would get
stuck INSIDE that call permanently, and would never even reach the
`<-ctx.Done()` line below it — the code that watches for the shutdown
signal would be unreachable. The wrapper isn't there to "avoid blocking"
in general; it moves ONE particular permanent block (serving) out of
main's way, freeing main to do the block that actually matters: waiting
for the signal to stop.

### The same shape for a worker pool

Steps 2–4 by hand, with tools you already own:

- Stop intake: **close the jobs channel** (Module 4: drain-and-stop).
- Drain: wait on the pool's WaitGroup — but with a time limit, which is
  the `withTimeout` trick from Module 3's tests: wait in a goroutine
  that closes a channel, select on that channel vs a timer.
- Exit: log loudly if the deadline won — those abandoned jobs are the
  "payments in unknown state" story.

Your shippio `Runner.Shutdown` is exactly this. After this module's
exercises, reread it — it will look obvious.

▶ **Run:** `go run ./05-context/demo/04-shutdown` — a real HTTP server
with a slow endpoint, shut down mid-request. The in-flight request
finishes; new ones are refused; then the process exits.

## 7. Interview questions

Answer these out loud at review time. Precise beats fast.

1. What three things does a context carry? Explain the tree: what does
   canceling a parent do, and why doesn't cancellation flow upward?
2. Why must you `defer cancel()` even for a WithTimeout context that
   cancels itself? What exactly leaks if you don't?
3. `context.Canceled` vs `context.DeadlineExceeded` — how does each
   happen, and why would production code treat them differently? What do
   WithCancelCause/Cause add?
4. Your function takes ctx but runs a pure-CPU loop for 30 seconds. Why
   does cancellation not affect it, and what are the three ways code
   actually LISTENS to a context?
5. errgroup vs WaitGroup: name the three things errgroup adds. When one
   goroutine in a group fails, what happens to the others — and what must
   THEY do for that to work?
6. Walk the four steps of graceful shutdown in order. Why is the drain
   context derived from Background instead of the signal context? Why
   does the drain need its own deadline at all?
7. `context.WithValue`: what belongs in it, what never does, and why is
   hiding a function parameter in a context worse than an explicit
   argument?
8. Module 4's done channel and ctx.Done() — same idea. What does context
   add that made it the universal standard, and name a real API where the
   library cancels I/O for you when ctx dies.

## 8. This module's files

```
demo/01-tree/        three-level context tree, canceled in the middle
demo/02-timeout/     WithTimeout racing slow work; both Err() outcomes
demo/03-errgroup/    one failure cancels the rest, timed proof
demo/04-shutdown/    real HTTP server, SIGTERM-style drain, mid-request
exercises/           YOUR work — see exercises/README.md
```

```powershell
go run ./05-context/demo/01-tree            # demos
go test ./05-context/...                    # fast native runs
.\test-race.ps1 ./05-context/...            # definition of done
```
