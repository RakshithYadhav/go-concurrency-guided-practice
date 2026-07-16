# OPS-1743: Nightly ERP import exceeds its batch window

| Field | Value |
|-------|-------|
| Type | Bug (Performance) |
| Priority | P2 — High |
| Reporter | Support Lead |
| Assignee | Rakshith |
| Component | import-service |
| Labels | performance, batch-jobs, erp-integration |

## Description

The nightly order/BOM import currently completes at approximately
9:40 AM. It previously completed by approximately 2:30 AM. Planners
require current schedule data at 8:00 AM. Import volume doubled this
season and is expected to grow further.

The import enriches each order line with lead time and unit cost from
the customer's ERP API. Calls are made sequentially, one at a time.
Each call takes 20–30 ms (WAN latency to the customer's system).

Vendor contract: maximum **32 concurrent connections** per customer.
Calls above the limit are **rejected**, not queued. Repeated
violations risk account suspension.

## Rakshith's Understanding

The customer's ERP is the system of record for sales orders — orders
originate there, not in our product. Each night, the import already
has the order feed (`SKU` and `Qty` per line, from the customer's
ERP). For each order line, a separate call is made to the ERP to fetch
`LeadDays` and `UnitCost` — this is the enrichment step described
above. These calls are made sequentially: one call per order line, and
each call completes before the next one starts. This was confirmed
correct.

## Steps to Reproduce

1. Run the scaled reproduction:
   ```powershell
   go run ./scenarios/01-erp-import/measure
   ```
   300 records represent the ~250k-row production feed. Per-call
   latency, the connection cap, and the code structure match
   production.
2. Compare the reported wall time against the 2-second target.

## Expected Result

- 300 records import in under 2 seconds
- 0 rejected ERP calls
- Output preserves input order; every record enriched correctly

## Actual Result

- 300 records take ~7.6 seconds (3.8× over target)
- 0 rejected calls (the sequential code never opens more than 1
  connection)

## Impact

- Planners work from previous-day schedule data until ~9:40 AM;
  requirement is 8:00 AM
- 14 support tickets this month, 2 escalations

## Environment

- Scaled reproduction in this repository
- `erp.go` simulates the vendor's system — **do not modify**
- The fix belongs in `import.go` (the import service)

## Acceptance Criteria

- [ ] Baseline measurement recorded in `RESULT.md` before any code
      change
- [ ] Import completes in under 2 seconds with 0 rejected calls and
      input order preserved (asserted by `import_test.go`)
- [ ] Tests pass natively and under `.\test-race.ps1 ./scenarios/...`
- [ ] `RESULT.md` contains before/after measurements
- [ ] Entry written to `WAR-STORIES.md`

## Notes

Do not drop or reorder records — finance reconciles against this
output.

## Additional Information

**Who is the planner?** A planner is a human — an employee at the
customer (a manufacturing company) who schedules production. They
decide which work order runs on which machine and in what sequence,
and use the Gantt view in our product to see and adjust that schedule.
Planner is a job role, not a service or a system.

**What data is being imported?** Order and BOM (Bill of Materials)
data from the customer's ERP. Specifically: new or changed sales
orders that need to become production work orders, plus the routing
details attached to each order line — which parts are needed, from
which supplier, at what lead time, and at what unit cost. This is the
`LeadDays` and `UnitCost` data returned by `FetchDetails` in `erp.go`.

**Do planners run the import, like a cron job?** No. The import is a
backend batch job that runs on a schedule, unattended, on our side —
planners never trigger it or see it running. Their only interaction
with it is indirect: they expect the schedule to reflect the previous
night's ERP data when they log in each morning.

**What is the ERP system?** ERP stands for Enterprise Resource
Planning. It is the customer's own software for running their core
business operations — sales orders, inventory, purchasing,
bill-of-materials data, and often finance and HR. Common ERP products
include SAP, Oracle NetSuite, Microsoft Dynamics, and Infor.

The ERP is not part of our product. Each customer runs their own ERP
separately, as their system of record for orders and materials. This
import service exists because our scheduling product needs that data
to build an accurate schedule. Every night, the import calls the
customer's ERP over its API and pulls the current order/BOM data into
our system, so the Gantt view reflects what is actually in the
customer's ERP.

**Why are `LeadDays` and `UnitCost` fetched separately from the base
order data, instead of in the same call?** (Likely interview question.)

- **The two kinds of data live in different parts of the ERP.** `SKU`
  and `Qty` are transactional data, owned by the order itself. Lead
  time and cost are supplier/item data, owned by the ERP's item master
  or supplier records — usually maintained by a different team
  (purchasing, not sales) and exposed through a different part of the
  ERP's API. Most ERPs do not join these together in one response.
- **Lead time and cost change independently of the order.** A supplier
  delay or a price change can happen after the order was placed. If
  the ERP stored `LeadDays`/`UnitCost` permanently on the order, that
  data would go stale. Fetching it at import time means scheduling
  uses current values, not whatever was true when the order was
  created.
- **The vendor treats the two as different classes of API traffic.**
  Pulling the order feed is a bulk, low-cost operation. Looking up
  per-item supplier/cost data is a more expensive, individual lookup —
  which is why it is the one with 20-30ms latency and the 32-connection
  cap in this ticket. The vendor is protecting that specific lookup,
  not the order feed as a whole.
