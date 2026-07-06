package exercises

import (
	"fmt"
	"testing"
)

// Heavy contention (300 records, ~half failing) makes lost appends show up
// reliably even without -race — same reasoning as ex2's WordFrequency test.
func TestValidateAll_Count(t *testing.T) {
	var records []string
	wantErrs := 0
	for i := 0; i < 300; i++ {
		records = append(records, fmt.Sprintf("record-%d", i))
		if i%2 == 0 {
			wantErrs++
		}
	}

	validate := func(r string) error {
		var i int
		fmt.Sscanf(r, "record-%d", &i)
		if i%2 == 0 {
			return fmt.Errorf("%s: invalid", r)
		}
		return nil
	}

	for iter := 0; iter < 20; iter++ {
		errs := ValidateAll(records, validate)
		if len(errs) != wantErrs {
			t.Fatalf("iter %d: got %d errors, want %d (lost appends — concurrent writers?)", iter, len(errs), wantErrs)
		}
	}
}

func TestValidateAll_Empty(t *testing.T) {
	errs := ValidateAll(nil, func(string) error { return nil })
	if len(errs) != 0 {
		t.Fatalf("got %d errors for no records, want 0", len(errs))
	}
}

func TestValidateAll_AllValid(t *testing.T) {
	records := []string{"a", "b", "c"}
	errs := ValidateAll(records, func(string) error { return nil })
	if len(errs) != 0 {
		t.Fatalf("got %d errors, want 0 (all records were valid)", len(errs))
	}
}
