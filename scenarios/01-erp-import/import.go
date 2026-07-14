package erpimport

// import.go — YOUR service. This is the file you fix.
//
// This is the import exactly as it runs in production today: one ERP
// call at a time, in a loop. It is correct. It is also why planners
// see yesterday's schedule — see PROBLEM.md for the ticket, the
// target, and the constraints.

// ImportOrders enriches every record against the ERP and returns the
// results in INPUT ORDER: out[i] belongs to records[i]. On any ERP
// error it returns that error.
func ImportOrders(records []OrderRecord, erp *ERPClient) ([]EnrichedOrder, error) {
	out := make([]EnrichedOrder, 0, len(records))

	for _, r := range records {
		enriched, err := erp.FetchDetails(r)
		if err != nil {
			return nil, err
		}
		out = append(out, enriched)
	}

	return out, nil
}
