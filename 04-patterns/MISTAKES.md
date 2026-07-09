# Mistakes Log — Module 4: Patterns I (Pipelines & Pools)

Every real bug made while solving this module's exercises — what was
written, why it broke, what fixed it. Same format as Modules 1-3.
(Patterns worth rereading first: forgetting `close` on a stage's output,
multi-sender close without the WaitGroup+closer, cleanup skipped by early
returns, and the defer-right-after-Lock rule — the same shapes reappear
here wearing pipeline clothes.)

---

## Exercise 1 — Pipeline stages, 2026-07-09

**Attempt 1 — ranging over a slice like a channel:**
```go
for num := range nums {   // nums is a []int, not a channel
    out <- num
}
```
Ranging over a **slice** with one loop variable gives you the **index**,
not the value — the opposite of ranging over a **channel**, where the
single variable IS the value (used correctly in `Square` and `Keep`,
both of which range over `in`, a channel). `Generate(3, 1, 4, 1, 5)` sent
`0, 1, 2, 3, 4` instead. Fixed with `for _, num := range nums`. Worth
remembering: slice range needs `_, v` to skip the index; channel range
never has an index to skip.

## Exercise 2 — Worker pool, 2026-07-09

The pool machinery itself (jobs channel, N workers, results channel,
WaitGroup + closer goroutine) was correct on the first pass. Two separate
bugs on top of it:

**Bug 1 — length vs capacity, third occurrence of this exact pattern**
(first two: Module 1 `ValidateAll`):
```go
output := make([]int, len(jobs))   // length 10, ten zeros already in it
...
output = append(output, res)        // appends AFTER those 10 zeros
```
10 jobs produced 20 results: 10 leftover zeros followed by the 10 real
ones. Fixed with `make([]int, 0, len(jobs))` — capacity reserved, length
zero, so `append` fills it from the start.

**Bug 2 — slice range sending indices, same shape as ex1's bug, one
exercise later:**
```go
for j := range jobs {   // jobs is []int — j is the INDEX
    jbs <- j
}
```
`ProcessAll([]int{7}, 8, v+1)` returned `1` instead of `8` — `fn` was
called with the index `0`, not the value `7`. Fixed with
`for _, j := range jobs`. Two exercises in a row with the identical bug
shape (slice range needs `_, v`) — worth deliberately double-checking
every `range` over a slice for this from now on, not just channels.
