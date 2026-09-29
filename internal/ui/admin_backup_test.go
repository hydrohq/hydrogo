package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderAdminBackupRequiresPasswordConfirmation(t *testing.T) {
	var output bytes.Buffer
	if err := RenderAdminBackup(&output, "csrf-token", "nonce", "en", nil, false, "trusted-public-key"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/admin/backup", `name="_csrf" value="csrf-token"`, `name="password"`, "Create and download backup", "trusted-public-key"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("backup page missing %q", want)
		}
	}
}
