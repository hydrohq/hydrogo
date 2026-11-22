package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hydrohq/hydrogo/internal/auth"
	"github.com/hydrohq/hydrogo/internal/updater"
)

func TestUpgradeStatusApplyAndRollbackCLI(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pubB64 := base64.StdEncoding.EncodeToString(pub)

	newBin := []byte("hydro-new-binary-v9.9.9")
	sum := sha256.Sum256(newBin)
	sumHex := hex.EncodeToString(sum[:])
	platformKey := runtime.GOOS + "/" + runtime.GOARCH

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			manifest := updater.UpdateManifest{
				FormatVersion: 1,
				Version:       "v9.9.9",
				Edition:       "core",
				Assets: map[string]updater.Asset{
					platformKey: {
						URL:    srv.URL + "/hydro-v9.9.9",
						SHA256: sumHex,
						Size:   int64(len(newBin)),
					},
				},
			}
			raw, _ := json.Marshal(manifest)
			sig := ed25519.Sign(priv, raw)
			_ = json.NewEncoder(w).Encode(updater.SignedUpdateManifest{
				Manifest:  raw,
				PublicKey: pubB64,
				Signature: base64.StdEncoding.EncodeToString(sig),
			})
			return
		}
		_, _ = w.Write(newBin)
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	err = run([]string{
		"upgrade", "status",
		"--manifest-url", srv.URL + "/manifest.json",
		"--public-key", pubB64,
		"--json",
	}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("upgrade status failed: %v (%s)", err, stderr.String())
	}
	var st updater.UpdateStatus
	if err := json.Unmarshal(stdout.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if !st.UpdateAvailable || st.LatestVersion != "v9.9.9" {
		t.Fatalf("unexpected update status: %+v", st)
	}

	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	exePath := filepath.Join(dir, "hydro")
	if err := os.WriteFile(exePath, []byte("hydro-old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	err = run([]string{
		"upgrade", "apply",
		"--manifest-url", srv.URL + "/manifest.json",
		"--public-key", pubB64,
		"--executable", exePath,
		"--yes",
	}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("upgrade apply failed: %v (%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "upgrade-applied version=v9.9.9") {
		t.Fatalf("unexpected apply output: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = run([]string{
		"upgrade", "rollback",
		"--executable", exePath,
		"--yes",
	}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("upgrade rollback failed: %v (%s)", err, stderr.String())
	}
	got, err := os.ReadFile(exePath)
	if err != nil || string(got) != "hydro-old-binary" {
		t.Fatalf("rollback content=%q err=%v", string(got), err)
	}
}

func TestAdminSystemUpdatesHandler(t *testing.T) {
	app := &serveApp{}

	// Non-admin is forbidden
	editorRec := httptest.NewRecorder()
	app.adminSystemUpdatesHandler(editorRec, adminAPIRequest(http.MethodGet, "/api/v1/admin/system/updates", "", &auth.User{ID: "e1", Role: auth.RoleEditor, Active: true}))
	if editorRec.Code != http.StatusForbidden {
		t.Fatalf("editor status=%d, want 403", editorRec.Code)
	}

	// Admin receives runtime mode & current version
	adminRec := httptest.NewRecorder()
	app.adminSystemUpdatesHandler(adminRec, adminAPIRequest(http.MethodGet, "/api/v1/admin/system/updates", "", &auth.User{ID: "a1", Role: auth.RoleAdmin, Active: true}))
	if adminRec.Code != http.StatusOK {
		t.Fatalf("admin status=%d, want 200", adminRec.Code)
	}
	if !strings.Contains(adminRec.Body.String(), `"runtime_mode"`) {
		t.Fatalf("missing runtime_mode in response: %s", adminRec.Body.String())
	}
}
