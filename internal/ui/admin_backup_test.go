package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderAdminBackupRequiresPasswordConfirmation(t *testing.T) {
	var output bytes.Buffer
	if err := RenderAdminBackup(&output, "csrf-token", "en", nil, false, "trusted-public-key"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/admin/backup", `name="_csrf" value="csrf-token"`, `name="password"`, "Create and download backup", "trusted-public-key"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("backup page missing %q", want)
		}
	}
}

func TestRenderAdminBackupLocalizesTurkishPage(t *testing.T) {
	var output bytes.Buffer
	if err := RenderAdminBackup(&output, "csrf-token", "tr", nil, false, "trusted-public-key"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Tam site yedeği", "Devam etmek için parolanızı doğrulayın", "Yedeği oluştur ve indir", "Bulut üzerinden zamanlanmış yedekleme ayarlanmamış."} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("Turkish backup page missing %q", want)
		}
	}
	if strings.Contains(output.String(), "Full site backup") || strings.Contains(output.String(), "Create and download backup") {
		t.Fatal("Turkish backup page includes English copy")
	}
}
