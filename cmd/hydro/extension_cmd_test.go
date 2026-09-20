package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	extensionv1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/licensing"
)

func TestExtensionCLIStagedLifecycle(t *testing.T) {
	root := t.TempDir()
	assets := map[string][]byte{"assets/app.js": []byte("console.log('acme')")}
	manifest := extensionv1.AdminExtensionManifest{
		ID: "acme-sales", Name: "Acme Sales", Version: "1.0.0",
		EntrypointURL: "https://reports.example/admin",
		Surfaces:      []extensionv1.AdminSurface{{ID: "sales", Label: "Sales", Path: "/sales", Placement: "commerce", Permissions: []string{"reports.sales.read"}}},
	}
	files, err := extensionv1.HashFiles(assets)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := extensionv1.SignPackage(extensionv1.PackageMetadata{FormatVersion: extensionv1.ExtensionPackageFormatVersion, Manifest: manifest, Files: files}, "vendor-test", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	packagePath := filepath.Join(root, "acme-sales.hx")
	if err := extensionv1.WriteArchive(packagePath, signed, assets); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(root, "trusted.pub")
	if err := os.WriteFile(keyPath, []byte(licensing.EncodePublicKey(publicKey)), 0o600); err != nil {
		t.Fatal(err)
	}
	staging := filepath.Join(root, "staging")
	runCLI := func(args ...string) string {
		t.Helper()
		var stdout, stderr bytes.Buffer
		if err := run(append([]string{"extension"}, args...), strings.NewReader(""), &stdout, &stderr); err != nil {
			t.Fatalf("args=%v err=%v stderr=%s", args, err, stderr.String())
		}
		return stdout.String()
	}
	runCLI("install", "--package", packagePath, "--staging", staging, "--trusted-key", keyPath)
	runCLI("approve", "--id", manifest.ID, "--staging", staging, "--all-requested")
	if output := runCLI("enable", "--id", manifest.ID, "--staging", staging); !strings.Contains(output, "state=enabled") {
		t.Fatalf("enable output = %s", output)
	}
	runCLI("suspend", "--id", manifest.ID, "--staging", staging)
	runCLI("revoke", "--id", manifest.ID, "--staging", staging)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"extension", "enable", "--id", manifest.ID, "--staging", staging}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("revoked extension was enabled")
	}
}
