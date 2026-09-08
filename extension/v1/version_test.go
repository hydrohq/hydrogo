package v1

import "testing"

func TestCandidateVersionIsExplicit(t *testing.T) {
	if APIVersion != "1.0.0-candidate" {
		t.Fatalf("API version = %q", APIVersion)
	}
}
