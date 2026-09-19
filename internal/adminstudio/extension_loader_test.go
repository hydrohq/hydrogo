package adminstudio

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	extensionv1 "github.com/ardaozyildirim/hydrogo/extension/v1"
)

func TestLoadSignedAdminExtensions(t *testing.T) {
	dir := t.TempDir()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keysPath := filepath.Join(dir, "trusted-keys.json")
	keysData, _ := json.Marshal(map[string]string{"vendor": base64.RawStdEncoding.EncodeToString(publicKey)})
	if err := os.WriteFile(keysPath, keysData, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := extensionv1.AdminExtensionManifest{ID: "reports", Name: "Reports", Version: "1", EntrypointURL: "https://reports.example", Surfaces: []extensionv1.AdminSurface{{ID: "home", Label: "Home", Path: "/", Placement: "reports"}}}
	signed, err := extensionv1.SignAdminExtensionManifest(manifest, "vendor", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(signed)
	if err := os.WriteFile(filepath.Join(dir, "reports.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	registry, err := LoadSignedAdminExtensions(dir, keysPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := registry.Manifests(); len(got) != 1 || got[0].ID != "reports" {
		t.Fatalf("loaded manifests = %#v", got)
	}
}

func TestLoadSignedAdminExtensionsRequiresTrustedKeys(t *testing.T) {
	if _, err := LoadSignedAdminExtensions(t.TempDir(), ""); err == nil {
		t.Fatal("expected trusted key requirement")
	}
}
