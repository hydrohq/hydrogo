package v1

import "testing"

func TestCandidateVersionIsExplicit(t *testing.T) {
	if APIVersion != "1.0.0-candidate" {
		t.Fatalf("API version = %q", APIVersion)
	}
}

func TestSDKCompatibilityUsesSameMajorRule(t *testing.T) {
	if err := CompatibleWith("1.2.0", APIVersion); err != nil {
		t.Fatal(err)
	}
	if err := CompatibleWith("2.0.0", APIVersion); err == nil {
		t.Fatal("different major accepted")
	}
	if _, err := Major("candidate"); err == nil {
		t.Fatal("invalid SDK version accepted")
	}
}
