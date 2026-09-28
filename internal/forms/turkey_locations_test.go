package forms

import "testing"

func TestTurkishLocationFieldsContainAllProvincesAndDistricts(t *testing.T) {
	fields, err := TurkishLocationFields()
	if err != nil {
		t.Fatal(err)
	}
	districtCount := 0
	for _, options := range fields[1].CascadeOptions {
		districtCount += len(options)
	}
	if len(fields) != 2 || len(fields[0].Options) != 81 || len(fields[1].CascadeOptions) != 81 || districtCount != 922 {
		t.Fatalf("incomplete bundled location data: provinces=%d province maps=%d district records=%d", len(fields[0].Options), len(fields[1].CascadeOptions), districtCount)
	}
	form := Form{ID: "address", Title: "Adres", Fields: fields}
	if err := form.Validate(); err != nil {
		t.Fatalf("location field contract is invalid: %v", err)
	}
	if _, errs, err := form.NormalizeSubmission(map[string]string{"province": "İstanbul", "district": "Çankaya"}); err != nil || len(errs) == 0 {
		t.Fatalf("district from another province accepted: errors=%v err=%v", errs, err)
	}
}
