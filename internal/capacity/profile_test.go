package capacity

import "testing"

func TestProfileDefaultsAreBounded(t *testing.T) {
	for _, profile := range []Profile{Compact, Standard, Distributed} {
		budget, err := Defaults(profile)
		if err != nil {
			t.Fatal(err)
		}
		if err := budget.Validate(); err != nil {
			t.Fatalf("%s budget: %v", profile, err)
		}
	}
	if _, err := Defaults(Profile("unknown")); err == nil {
		t.Fatal("unknown profile accepted")
	}
}
