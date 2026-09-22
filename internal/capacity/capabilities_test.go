package capacity

import "testing"

func TestManifestIsStableAndProfileAware(t *testing.T) {
	compact, err := Manifest(Compact)
	if err != nil {
		t.Fatal(err)
	}
	standard, err := Manifest(Standard)
	if err != nil {
		t.Fatal(err)
	}
	if len(standard) <= len(compact) {
		t.Fatalf("standard manifest should expose additional bounded capabilities: compact=%d standard=%d", len(compact), len(standard))
	}
	for i := 1; i < len(compact); i++ {
		if compact[i-1].Name >= compact[i].Name {
			t.Fatalf("manifest is not sorted: %#v", compact)
		}
	}
	for _, item := range compact {
		if item.Name == "distributed-cache" || item.Name == "otel-export" {
			t.Fatalf("compact profile unexpectedly activates distributed capability %q", item.Name)
		}
	}
}
