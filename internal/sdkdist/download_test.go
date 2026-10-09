package sdkdist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/release"
	"github.com/ardaozyildirim/hydrogo/internal/testhttp"
)

func TestDownloadAndInstallLocalArtifact(t *testing.T) {
	payload := []byte("runtime-binary")
	digest := sha256.Sum256(payload)
	const endpoint = "http://127.0.0.1:19111"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19111": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})})
	root := t.TempDir()
	destination := filepath.Join(root, "cache", "hydro")
	expected := artifactForPayload("hydro-linux-amd64", payload)
	if err := DownloadAndInstall(context.Background(), endpoint, expected, destination, DownloadOptions{Client: client, AllowInsecureLocalhost: true}); err != nil {
		t.Fatalf("download and install: %v", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || string(got) != string(payload) {
		t.Fatalf("installed payload = %q/%v", got, err)
	}
	_ = digest
}

func TestDownloadRejectsOversizedResponseAndRedirect(t *testing.T) {
	const redirectURL = "http://127.0.0.1:19112"
	redirectClient := testhttp.Client(testhttp.Handlers{"127.0.0.1:19112": http.RedirectHandler("http://127.0.0.1:19112/runtime", http.StatusFound)})
	payload := []byte("runtime")
	expected := artifactForPayload("hydro-linux-amd64", payload)
	if err := DownloadAndInstall(context.Background(), redirectURL, expected, filepath.Join(t.TempDir(), "hydro"), DownloadOptions{Client: redirectClient, AllowInsecureLocalhost: true}); err == nil {
		t.Fatal("redirect response was accepted")
	}
	const oversizedURL = "http://127.0.0.1:19113"
	oversizedClient := testhttp.Client(testhttp.Handlers{"127.0.0.1:19113": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("too-large"))
	})})
	if err := DownloadAndInstall(context.Background(), oversizedURL, expected, filepath.Join(t.TempDir(), "hydro"), DownloadOptions{Client: oversizedClient, AllowInsecureLocalhost: true, MaxBytes: 4}); err == nil {
		t.Fatal("oversized response was accepted")
	}
}

func TestDownloadGrantValidatesBeforeInstall(t *testing.T) {
	payload := []byte("grant-runtime")
	const endpoint = "http://127.0.0.1:19114"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19114": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	})})
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	grant := ArtifactGrant{AccessToken: "token-123456", ProjectID: "proj-1", CoreVersion: "1.2.3", SDKVersion: "1.0.0", ArtifactURL: endpoint, IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
	expected := artifactForPayload("hydro-linux-amd64", payload)
	destination := filepath.Join(t.TempDir(), "runtime")
	if err := DownloadGrantAndInstall(context.Background(), grant, "proj-1", "1.2.3", "1.0.0", now, expected, destination, DownloadOptions{Client: client, AllowInsecureLocalhost: true}); err != nil {
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
