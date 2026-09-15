package adminstudio

import "testing"

func TestKitRegistryIsDeterministicAndOptional(t *testing.T) {
	r := NewKitRegistry(KitSearch, KitSEO, KitSearch)
	if !r.Enabled(KitSearch) || r.Enabled(KitCommerce) {
		t.Fatal("kit enablement incorrect")
	}
	got := r.Kits()
	if len(got) != 2 || got[0] != KitSEO || got[1] != KitSearch {
		t.Fatalf("kits = %v", got)
	}
}
