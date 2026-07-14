package erpimport

import (
	"testing"
	"time"
)

// The ticket's target, asserted: under 2s, zero rejections, input
// order, every record enriched correctly.
func TestImport_MeetsTicketTarget(t *testing.T) {
	records := MakeSampleRecords(300)
	erp := NewERPClient()

	start := time.Now()
	got, err := ImportOrders(records, erp)
	took := time.Since(start)

	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if rejected := erp.Rejected(); rejected > 0 {
		t.Fatalf("%d ERP calls REJECTED — the 32-connection contract was violated; bound your concurrency", rejected)
	}
	if len(got) != len(records) {
		t.Fatalf("want %d enriched records, got %d — no dropped records, finance reconciles against this", len(records), len(got))
	}
	for i, r := range records {
		if got[i] != enrichmentFor(r) {
			t.Fatalf("got[%d] = %+v, want %+v — results must stay in INPUT ORDER, matched to their record", i, got[i], enrichmentFor(r))
		}
	}
	if took > 2*time.Second {
		t.Fatalf("300 records took %v — the ticket's target is under 2s. One call at a time can't get there; PROBLEM.md has the constraint that shapes the fix", took)
	}
}

func TestImport_EmptyFeed(t *testing.T) {
	got, err := ImportOrders(nil, NewERPClient())
	if err != nil {
		t.Fatalf("empty feed: want nil error, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("empty feed: want 0 results, got %d", len(got))
	}
}
