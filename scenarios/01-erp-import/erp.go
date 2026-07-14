package erpimport

// erp.go — the customer's ERP system, simulated. ⛔ DO NOT MODIFY.
//
// This file stands in for the ERP vendor's API. You cannot change the
// vendor's system in real life, so you don't get to change this file
// either. Two properties are contractual and real:
//
//   - each call takes 20–30ms (WAN latency to the customer's ERP)
//   - at most 32 calls may be in flight at once; the 33rd is REJECTED
//     with an error, exactly like the real vendor's connection limit.
//     It does not queue. It does not retry for you.

import (
	"errors"
	"math/rand/v2"
	"sync/atomic"
	"time"
)

// MaxConcurrentERP is the vendor's contractual connection cap.
const MaxConcurrentERP = 32

// ErrTooManyConnections is what the vendor returns when the cap is
// exceeded. If you see this, YOUR side is over-fanning.
var ErrTooManyConnections = errors.New(
	"ERP rejected the call: more than 32 concurrent connections (vendor contract limit) — bound your concurrency")

// OrderRecord is one line of the nightly feed.
type OrderRecord struct {
	SKU string
	Qty int
}

// EnrichedOrder is an order line with the ERP's data attached.
type EnrichedOrder struct {
	SKU      string
	Qty      int
	LeadDays int // supplier lead time
	UnitCost int // cents
}

// ERPClient is the connection to one customer's ERP.
type ERPClient struct {
	inFlight atomic.Int32
	calls    atomic.Int64
	rejected atomic.Int64
}

// NewERPClient returns a client for one customer's ERP.
func NewERPClient() *ERPClient { return &ERPClient{} }

// FetchDetails looks up lead time and unit cost for one order line.
// It takes 20–30ms and enforces the 32-connection contract.
func (c *ERPClient) FetchDetails(r OrderRecord) (EnrichedOrder, error) {
	c.calls.Add(1)
	n := c.inFlight.Add(1)
	defer c.inFlight.Add(-1)

	if n > MaxConcurrentERP {
		c.rejected.Add(1)
		return EnrichedOrder{}, ErrTooManyConnections
	}

	time.Sleep(time.Duration(20+rand.IntN(11)) * time.Millisecond)
	return enrichmentFor(r), nil
}

// Calls reports how many times FetchDetails was invoked.
func (c *ERPClient) Calls() int64 { return c.calls.Load() }

// Rejected reports how many calls broke the 32-connection contract.
func (c *ERPClient) Rejected() int64 { return c.rejected.Load() }

// enrichmentFor is the ERP's (deterministic) answer for a record. The
// tests use it to check correctness without touching the network.
func enrichmentFor(r OrderRecord) EnrichedOrder {
	return EnrichedOrder{
		SKU:      r.SKU,
		Qty:      r.Qty,
		LeadDays: 1 + len(r.SKU)%7,
		UnitCost: 100*len(r.SKU) + r.Qty,
	}
}

// MakeSampleRecords builds the scaled-down nightly feed used by both
// the tests and the measurement harness.
func MakeSampleRecords(n int) []OrderRecord {
	records := make([]OrderRecord, n)
	for i := range records {
		records[i] = OrderRecord{
			SKU: "SKU-" + string(rune('A'+i%26)) + "-" + itoa(i),
			Qty: 1 + i%40,
		}
	}
	return records
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
