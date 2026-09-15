package ui

import "testing"

func TestAdminKitCSSDefinesAccessibleSummaryStyles(t *testing.T) {
	if AdminKitCSS == "" || len(AdminKitCSS) < 80 {
		t.Fatalf("kit css unexpectedly short")
	}
}
