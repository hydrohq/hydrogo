package adminstudio

import "testing"

func TestKitRegistryIsDeterministicAndOptional(t *testing.T) {
	r := NewKitRegistry(KitSearch, KitSEO, KitSearch)
	if !r.Enabled(KitSearch) || r.Enabled(KitCommerce) {
		t.Fatal("kit enablement incorrect")
	}
	got := r.Kits()
	if len(got) != 2 || got[0] != KitSearch || got[1] != KitSEO {
		t.Fatalf("kits = %v", got)
	}
}

func TestUnknownKitIsRejected(t *testing.T) {
	if IsKnownKit(Kit("payments-provider-internals")) {
		t.Fatal("unknown kit accepted")
	}
}

func TestValidateKitsRejectsDuplicates(t *testing.T) {
	if err := ValidateKits([]Kit{KitSEO, KitSEO}); err != ErrInvalidKitSet {
		t.Fatalf("duplicate kits error = %v", err)
	}
}
