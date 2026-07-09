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

Recall Module 1's `FetchAll` shape: WaitGroup + slots. Now add two
production requirements: **collect the first error**, and **when one
fails, stop the others instead of letting them burn time on a doomed
request**. Hand-wiring that takes a WaitGroup, an error channel or
mutexed error variable, and a cancel — every time. `errgroup` (package
`golang.org/x/sync/errgroup`) packages it:

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

What `errgroup.WithContext` promises:

1. `g.Go(f)` runs `f` in a goroutine (Add/Done handled for you).
2. `g.Wait()` blocks until all return — and returns the **first**
   non-nil error (later errors are dropped, deliberately: the first one
   is almost always the root cause).
3. **The moment any `f` returns an error, the group's `ctx` is
   canceled** — every other `f` that respects ctx stops early. That's
   the whole trick: first failure pulls the plug on the rest.
4. Bonus: `g.SetLimit(n)` caps how many run at once — Module 4's
   bounded parallelism, one method call.

The mental model: **WaitGroup for goroutines that can't fail; errgroup
for goroutines that can.** In production, they can.

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
