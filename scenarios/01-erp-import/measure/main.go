// Measurement harness for OPS-1743. Run this FIRST, before changing
// any code, and write what it prints into RESULT.md as your baseline.
// Run it again after your fix — that's your "after."
//
//	go run ./scenarios/01-erp-import/measure
package main

import (
	"fmt"
	"time"

	erpimport "github.com/rakshith/go-concurrency/scenarios/01-erp-import"
)

func main() {
	const n = 300
	records := erpimport.MakeSampleRecords(n)
	erp := erpimport.NewERPClient()

	fmt.Printf("importing %d records (ERP: 20-30ms/call, hard cap %d concurrent)...\n\n",
		n, erpimport.MaxConcurrentERP)

	start := time.Now()
	out, err := erpimport.ImportOrders(records, erp)
	took := time.Since(start)

	fmt.Printf("  wall time:     %v\n", took.Round(time.Millisecond))
	fmt.Printf("  throughput:    %.0f records/sec\n", float64(len(out))/took.Seconds())
	fmt.Printf("  ERP calls:     %d\n", erp.Calls())
	fmt.Printf("  ERP rejected:  %d\n", erp.Rejected())
	if err != nil {
		fmt.Printf("  import FAILED: %v\n", err)
	}

	fmt.Println()
	const target = 2 * time.Second
	switch {
	case err != nil || erp.Rejected() > 0:
		fmt.Printf("result: FAILED — %d calls rejected (over the %d-connection limit)\n",
			erp.Rejected(), erpimport.MaxConcurrentERP)
	case took > target:
		fmt.Printf("result: misses the %v target by %.1fx\n", target, float64(took)/float64(target))
	default:
		fmt.Printf("result: meets the %v target. Record this in RESULT.md.\n", target)
	}
}
