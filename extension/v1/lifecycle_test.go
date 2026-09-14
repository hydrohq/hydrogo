package v1

import "testing"

func TestLifecycleContractsValidateRequestsAndPlans(t *testing.T) {
	if err := ValidateLifecycleRequest(LifecycleRequest{Module: "pages", ToVersion: "2.0.0"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMigrationPlan(MigrationPlan{MigrationIDs: []string{"001", "002"}}); err != nil {
		t.Fatal(err)
	}
	for _, request := range []LifecycleRequest{{ToVersion: "1"}, {Module: "pages"}, {Module: "pages", FromVersion: "1", ToVersion: "1"}} {
		if err := ValidateLifecycleRequest(request); err == nil {
			t.Fatalf("invalid lifecycle request accepted: %#v", request)
		}
	}
	if err := ValidateMigrationPlan(MigrationPlan{MigrationIDs: []string{"001", "001"}}); err == nil {
		t.Fatal("duplicate migration plan ID accepted")
	}
}
