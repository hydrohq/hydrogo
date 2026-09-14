package ui

import "testing"

type testDesignSystem struct{ name string }

func (p testDesignSystem) Name() string            { return p.name }
func (testDesignSystem) ButtonClass(string) string { return "btn" }
func (testDesignSystem) FieldClass() string        { return "field" }
func (testDesignSystem) TableClass() string        { return "table" }

func TestDesignSystemRegistryResolvesProvidersCaseInsensitive(t *testing.T) {
	r, err := NewDesignSystemRegistry(SemanticDesignSystem{}, testDesignSystem{name: "material"})
	if err != nil {
		t.Fatal(err)
	}
	provider, err := r.Resolve("MATERIAL")
	if err != nil || provider.Name() != "material" {
		t.Fatalf("provider=%v err=%v", provider, err)
	}
}

func TestDesignSystemRegistryRejectsDuplicateAndMissingProviders(t *testing.T) {
	if _, err := NewDesignSystemRegistry(testDesignSystem{name: "shadcn"}, testDesignSystem{name: "SHADCN"}); err == nil {
		t.Fatal("duplicate accepted")
	}
	if _, err := NewDesignSystemRegistry(testDesignSystem{}); err == nil {
		t.Fatal("unnamed provider accepted")
	}
	if _, err := (&DesignSystemRegistry{}).Resolve("missing"); err == nil {
		t.Fatal("missing provider resolved")
	}
}
