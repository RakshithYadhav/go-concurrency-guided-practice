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
