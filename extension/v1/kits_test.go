package v1

import "testing"

func TestCapabilityKitResolutionIsDeterministic(t *testing.T) {
	got, err := ResolveCapabilityKits([]CapabilityKit{KitCommerce, KitSEO})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != KitCommerce || got[1] != KitForms || got[2] != KitSEO {
		t.Fatalf("resolved kits = %#v", got)
	}
}
func TestCapabilityKitResolutionRejectsUnknownAndDuplicate(t *testing.T) {
	if _, err := ResolveCapabilityKits([]CapabilityKit{"unknown"}); err == nil {
		t.Fatal("expected unknown kit rejection")
	}
	if _, err := ResolveCapabilityKits([]CapabilityKit{KitSEO, KitSEO}); err == nil {
		t.Fatal("expected duplicate kit rejection")
	}
}
