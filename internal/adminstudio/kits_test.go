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

func TestResolveKitsAddsDependenciesDeterministically(t *testing.T) {
	got, err := ResolveKits([]Kit{KitCommerce, KitSEO})
	if err != nil || len(got) != 3 || got[0] != KitCommerce || got[1] != KitForms || got[2] != KitSEO {
		t.Fatalf("resolved kits = %v, err=%v", got, err)
	}
}

func TestNewResolvedKitRegistryValidatesAndExpands(t *testing.T) {
	registry, err := NewResolvedKitRegistry(KitCommerce)
	if err != nil {
		t.Fatal(err)
	}
	if !registry.Enabled(KitCommerce) || !registry.Enabled(KitForms) {
		t.Fatalf("resolved registry = %#v", registry.Kits())
	}
	if _, err := NewResolvedKitRegistry(Kit("unknown")); err != ErrInvalidKitSet {
		t.Fatalf("unknown kit error = %v", err)
	}
}

func TestDescribeKitsIsDeterministicAndCarriesDependencies(t *testing.T) {
	descriptors, err := DescribeKits([]Kit{KitSEO, KitCommerce})
	if err != nil || len(descriptors) != 3 {
		t.Fatalf("descriptors = %#v, err=%v", descriptors, err)
	}
	if descriptors[0].Name != KitCommerce || len(descriptors[0].Dependencies) != 1 || descriptors[0].Dependencies[0] != KitForms || !descriptors[0].Optional {
		t.Fatalf("commerce descriptor = %#v", descriptors[0])
	}
}
