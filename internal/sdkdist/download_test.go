package sdkdist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/release"
)

func TestDownloadAndInstallLocalArtifact(t *testing.T) {
	payload := []byte("runtime-binary")
	digest := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	root := t.TempDir()
	destination := filepath.Join(root, "cache", "hydro")
	expected := artifactForPayload("hydro-linux-amd64", payload)
	if err := DownloadAndInstall(context.Background(), server.URL, expected, destination, DownloadOptions{AllowInsecureLocalhost: true}); err != nil {
		t.Fatalf("download and install: %v", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || string(got) != string(payload) {
		t.Fatalf("installed payload = %q/%v", got, err)
	}
	_ = digest
}

func TestDownloadRejectsOversizedResponseAndRedirect(t *testing.T) {
	redirect := httptest.NewServer(http.RedirectHandler("http://127.0.0.1:1/runtime", http.StatusFound))
	defer redirect.Close()
	payload := []byte("runtime")
	expected := artifactForPayload("hydro-linux-amd64", payload)
	if err := DownloadAndInstall(context.Background(), redirect.URL, expected, filepath.Join(t.TempDir(), "hydro"), DownloadOptions{AllowInsecureLocalhost: true}); err == nil {
		t.Fatal("redirect response was accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("too-large"))
	}))
	defer server.Close()
	if err := DownloadAndInstall(context.Background(), server.URL, expected, filepath.Join(t.TempDir(), "hydro"), DownloadOptions{AllowInsecureLocalhost: true, MaxBytes: 4}); err == nil {
		t.Fatal("oversized response was accepted")
	}
}

func TestDownloadGrantValidatesBeforeInstall(t *testing.T) {
	payload := []byte("grant-runtime")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	grant := ArtifactGrant{AccessToken: "token-123456", ProjectID: "proj-1", CoreVersion: "1.2.3", SDKVersion: "1.0.0", ArtifactURL: server.URL, IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
	expected := artifactForPayload("hydro-linux-amd64", payload)
	destination := filepath.Join(t.TempDir(), "runtime")
	if err := DownloadGrantAndInstall(context.Background(), grant, "proj-1", "1.2.3", "1.0.0", now, expected, destination, DownloadOptions{AllowInsecureLocalhost: true}); err != nil {
		t.Fatalf("grant download and install: %v", err)
	}
	grant.ProjectID = "other"
	if err := DownloadGrantAndInstall(context.Background(), grant, "proj-1", "1.2.3", "1.0.0", now, expected, filepath.Join(t.TempDir(), "other"), DownloadOptions{AllowInsecureLocalhost: true}); err == nil {
		t.Fatal("mismatched grant was accepted")
	}
}

func artifactForPayload(name string, payload []byte) release.Artifact {
	digest := sha256.Sum256(payload)
	return release.Artifact{Path: name, Size: int64(len(payload)), SHA256: hex.EncodeToString(digest[:])}
}
