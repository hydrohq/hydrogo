package updater

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDetectRuntime(t *testing.T) {
	noFile := func(string) bool { return false }
	noEnv := func(string) string { return "" }

	if got := DetectRuntime(false, noEnv, noFile); got != RuntimeStandalone {
		t.Fatalf("got %q, want %q", got, RuntimeStandalone)
	}
	if got := DetectRuntime(true, noEnv, noFile); got != RuntimeCGI {
		t.Fatalf("got %q, want %q", got, RuntimeCGI)
	}
	if got := DetectRuntime(false, func(k string) string {
		if k == "GATEWAY_INTERFACE" {
			return "CGI/1.1"
		}
		return ""
	}, noFile); got != RuntimeCGI {
		t.Fatalf("got %q, want %q", got, RuntimeCGI)
	}
	if got := DetectRuntime(false, noEnv, func(p string) bool { return p == "/.dockerenv" }); got != RuntimeDocker {
		t.Fatalf("got %q, want %q", got, RuntimeDocker)
	}
}

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		cur, next string
		want      int
	}{
		{"0.5.1", "0.6.0", -1},
		{"v0.5.1", "v0.5.1", 0},
		{"0.6.0", "0.5.9", 1},
		{"dev", "0.6.0", -1},
	}
	for _, tc := range cases {
		if got := CompareSemver(tc.cur, tc.next); got != tc.want {
			t.Fatalf("CompareSemver(%q, %q) = %d, want %d", tc.cur, tc.next, got, tc.want)
		}
	}
}

func TestCheckApplyAndRollbackSignedUpdate(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	newBinary := []byte("#!/bin/sh\necho new-hydro-v0.6.0\n")
	sum := sha256.Sum256(newBinary)
	sumHex := hex.EncodeToString(sum[:])

	var srv *httptest.Server
	platformKey := runtime.GOOS + "/" + runtime.GOARCH
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			manifest := UpdateManifest{
				FormatVersion: 1,
				Version:       "v0.6.0",
				Edition:       "core",
				Notes:         "Security and performance release",
				Assets: map[string]Asset{
					platformKey: {
						URL:    srv.URL + "/hydro-v0.6.0",
						SHA256: sumHex,
						Size:   int64(len(newBinary)),
					},
				},
			}
			raw, _ := json.Marshal(manifest)
			sig := ed25519.Sign(priv, raw)
			signed := SignedUpdateManifest{
				Manifest:  raw,
				PublicKey: base64.StdEncoding.EncodeToString(pub),
				Signature: base64.StdEncoding.EncodeToString(sig),
			}
			_ = json.NewEncoder(w).Encode(signed)
		case "/hydro-v0.6.0":
			_, _ = w.Write(newBinary)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// 1. Check detects available update for core/standalone
	status, err := Check(context.Background(), srv.Client(), CheckOptions{
		CurrentVersion:   "v0.5.1",
		Edition:          "core",
		Runtime:          RuntimeStandalone,
		ManifestURL:      srv.URL + "/manifest.json",
		TrustedPublicKey: pub,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !status.UpdateAvailable || !status.CanSelfUpdate || status.LatestVersion != "v0.6.0" {
		t.Fatalf("unexpected status: %+v", status)
	}

	// 2. Edition mismatch must block self-update
	customStatus, err := Check(context.Background(), srv.Client(), CheckOptions{
		CurrentVersion:   "v0.5.1",
		Edition:          "beyond",
		Runtime:          RuntimeStandalone,
		ManifestURL:      srv.URL + "/manifest.json",
		TrustedPublicKey: pub,
	})
	if err != nil {
		t.Fatalf("Check(custom) failed: %v", err)
	}
	if customStatus.CanSelfUpdate {
		t.Fatal("expected CanSelfUpdate=false on edition mismatch")
	}

	// 3. Untrusted key must fail verification
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := Check(context.Background(), srv.Client(), CheckOptions{
		CurrentVersion:   "v0.5.1",
		Edition:          "core",
		Runtime:          RuntimeStandalone,
		ManifestURL:      srv.URL + "/manifest.json",
		TrustedPublicKey: otherPub,
	}); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("err = %v, want ErrInvalidSignature", err)
	}

	// 4. Apply replaces binary atomically and saves .prev; Rollback restores original
	dir, evalErr := filepath.EvalSymlinks(t.TempDir())
	if evalErr != nil {
		t.Fatal(evalErr)
	}
	exePath := filepath.Join(dir, "hydro")
	oldBinary := []byte("#!/bin/sh\necho old-hydro-v0.5.1\n")
	if err := os.WriteFile(exePath, oldBinary, 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Apply(context.Background(), srv.Client(), ApplyOptions{
		CheckOptions: CheckOptions{
			CurrentVersion:   "v0.5.1",
			Edition:          "core",
			Runtime:          RuntimeStandalone,
			ManifestURL:      srv.URL + "/manifest.json",
			TrustedPublicKey: pub,
		},
		ExecutablePath: exePath,
	})
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if res.Version != "v0.6.0" {
		t.Fatalf("applied version = %q, want v0.6.0", res.Version)
	}
	gotNew, err := os.ReadFile(exePath)
	if err != nil || string(gotNew) != string(newBinary) {
		t.Fatalf("executable not updated: err=%v content=%q", err, string(gotNew))
	}
	if err := Rollback(exePath); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}
	gotOld, err := os.ReadFile(exePath)
	if err != nil || string(gotOld) != string(oldBinary) {
		t.Fatalf("executable not rolled back: err=%v content=%q", err, string(gotOld))
	}
}

func TestApplyRejectsChecksumMismatchAndDocker(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	platformKey := runtime.GOOS + "/" + runtime.GOARCH
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			manifest := UpdateManifest{
				FormatVersion: 1,
				Version:       "v0.6.0",
				Edition:       "core",
				Assets: map[string]Asset{
					platformKey: {
						URL:    srv.URL + "/bin",
						SHA256: strings.Repeat("0", 64),
						Size:   4,
					},
				},
			}
			raw, _ := json.Marshal(manifest)
			sig := ed25519.Sign(priv, raw)
			_ = json.NewEncoder(w).Encode(SignedUpdateManifest{
				Manifest:  raw,
				PublicKey: base64.StdEncoding.EncodeToString(pub),
				Signature: base64.StdEncoding.EncodeToString(sig),
			})
			return
		}
		_, _ = w.Write([]byte("evil"))
	}))
	defer srv.Close()

	dir, evalErr := filepath.EvalSymlinks(t.TempDir())
	if evalErr != nil {
		t.Fatal(evalErr)
	}
	exePath := filepath.Join(dir, "hydro")
	_ = os.WriteFile(exePath, []byte("orig"), 0o755)

	_, err := Apply(context.Background(), srv.Client(), ApplyOptions{
		CheckOptions: CheckOptions{
			CurrentVersion:   "v0.5.1",
			Edition:          "core",
			Runtime:          RuntimeDocker,
			ManifestURL:      srv.URL + "/manifest.json",
			TrustedPublicKey: pub,
		},
		ExecutablePath: exePath,
	})
	if !errors.Is(err, ErrSelfUpdateUnsupported) {
		t.Fatalf("docker apply err = %v, want ErrSelfUpdateUnsupported", err)
	}

	_, err = Apply(context.Background(), srv.Client(), ApplyOptions{
		CheckOptions: CheckOptions{
			CurrentVersion:   "v0.5.1",
			Edition:          "core",
			Runtime:          RuntimeStandalone,
			ManifestURL:      srv.URL + "/manifest.json",
			TrustedPublicKey: pub,
		},
		ExecutablePath: exePath,
	})
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("checksum apply err = %v, want ErrChecksumMismatch", err)
	}
}

func TestCheckAndApplyRejectSDKIncompatible(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	platformKey := runtime.GOOS + "/" + runtime.GOARCH
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		manifest := UpdateManifest{
			FormatVersion: 1,
			Version:       "v0.6.0",
			Edition:       "core",
			MinSDKVersion: "1.2.0",
			MaxSDKVersion: "1.9.0",
			Assets: map[string]Asset{
				platformKey: {URL: srv.URL + "/bin", SHA256: strings.Repeat("a", 64), Size: 4},
			},
		}
		raw, _ := json.Marshal(manifest)
		sig := ed25519.Sign(priv, raw)
		_ = json.NewEncoder(w).Encode(SignedUpdateManifest{
			Manifest:  raw,
			PublicKey: base64.StdEncoding.EncodeToString(pub),
			Signature: base64.StdEncoding.EncodeToString(sig),
		})
	}))
	defer srv.Close()

	st, err := Check(context.Background(), srv.Client(), CheckOptions{
		CurrentVersion:   "v0.5.1",
		Edition:          "core",
		SDKVersion:       "1.0.0",
		Runtime:          RuntimeStandalone,
		ManifestURL:      srv.URL + "/manifest.json",
		TrustedPublicKey: pub,
	})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if st.CanSelfUpdate || !strings.Contains(st.Reason, "min_sdk_version") {
		t.Fatalf("expected CanSelfUpdate=false with min_sdk_version reason, got %+v", st)
	}

	_, err = Apply(context.Background(), srv.Client(), ApplyOptions{
		CheckOptions: CheckOptions{
			CurrentVersion:   "v0.5.1",
			Edition:          "core",
			SDKVersion:       "2.0.0",
			Runtime:          RuntimeStandalone,
			ManifestURL:      srv.URL + "/manifest.json",
			TrustedPublicKey: pub,
		},
		ExecutablePath: filepath.Join(t.TempDir(), "hydro"),
	})
	if !errors.Is(err, ErrSDKIncompatible) {
		t.Fatalf("Apply err = %v, want ErrSDKIncompatible", err)
	}
}
