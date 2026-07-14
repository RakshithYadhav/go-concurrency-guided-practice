# OPS-1743 — Nightly ERP import blows its window; planners start the day blind

**Reported by:** Support lead · **Assigned to:** you · **Priority:** HIGH

## Symptoms (production)

- The nightly order/BOM sync from customer ERPs used to finish around
  2:30am. Since onboarding two enterprise customers it finishes around
  **9:40am**.
- Planners open the Gantt at 8am and are looking at **yesterday's
  schedule**. 14 support tickets this month, 2 escalations.
- Feed volume doubled this season. It will grow again next season.
  Buying a faster machine is not a plan.

## What we know

- The import enriches each order line against the customer's ERP API
  (lead time + unit cost per SKU) — **one call at a time, in a loop**.
- Each ERP call takes **20–30ms** over the WAN. The math: volume ×
  25ms, serially. That's the whole outage.
- **Vendor contract: max 32 concurrent connections per customer.**
  Above that, the ERP **rejects** calls — it does not queue them.
  Getting the account suspended for hammering the vendor is not an
  option.

## This repo (scaled-down repro)

300 records stand in for the real ~250k-row feed. Same per-call
latency, same 32-connection cap, same code shape. The ratio is what
matters, not the absolute seconds.

**Before changing anything**, measure the baseline and write it into
`RESULT.md`:

```powershell
go run ./scenarios/01-erp-import/measure
```

## Target (the tests assert exactly this)

- 300 records imported in **under 2 seconds**
- **Zero** rejected ERP calls — the 32-connection contract holds
- Results in **input order**, every record enriched correctly

## Constraints

- `erp.go` simulates the vendor's system and is **DO NOT MODIFY** —
  you can't change the vendor, only your own service (`import.go`).
- No dropped or reordered records. Finance reconciles against this
  output.

## Definition of done

Tests green natively AND under `.\test-race.ps1 ./scenarios/...`,
`RESULT.md` filled with real before/after numbers, four-part package
written to `../../../WAR-STORIES.md`.
