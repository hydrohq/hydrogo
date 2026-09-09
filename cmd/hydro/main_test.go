package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	backupstore "github.com/ardaozyildirim/hydrogo/internal/backup"
	"github.com/ardaozyildirim/hydrogo/internal/release"
)

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"version"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("run version: %v", err)
	}
	if got := stdout.String(); got != version+"\n" {
		t.Fatalf("version output = %q, want %q", got, version+"\n")
	}
}

func TestRunVersionJSONIncludesSchemaMetadata(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"version", "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("run version --json: %v", err)
	}
	var info versionInfo
	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if info.Version != version || info.SchemaMigration == "" || info.GoVersion == "" {
		t.Fatalf("incomplete version info: %+v", info)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"unknown"}, strings.NewReader(""), &stdout, &stderr)
	if err == nil {
		t.Fatal("run unknown command succeeded")
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("usage missing from stderr: %q", stderr.String())
	}
}

func TestDeadLetterRequeueRequiresExplicitConfirmation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"jobs", "dead-letter", "requeue", "--id", "job-1"}, strings.NewReader(""), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("requeue without confirmation error = %v", err)
	}
}

func TestRunMigrateAndCreateAdmin(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	var stdout, stderr bytes.Buffer

	// 1. Run migrate
	err := run([]string{"migrate", "--database-dialect", "sqlite", "--database-dsn", dbPath}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("migrate failed: %v, stderr: %s", err, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"upgrade", "check", "--database-dialect", "sqlite", "--database-dsn", dbPath, "--capacity-profile", "compact"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("upgrade check failed: %v, stderr: %s", err, stderr.String())
	}
	if got := stdout.String(); got != "upgrade-ready profile=compact database=sqlite\n" {
		t.Fatalf("unexpected upgrade output: %q", got)
	}

	// 2. Run create-admin
	stdout.Reset()
	stderr.Reset()
	err = run([]string{
		"create-admin",
		"--email", "admin@example.com",
		"--password-stdin",
		"--database-dialect", "sqlite",
		"--database-dsn", dbPath,
	}, strings.NewReader("StrongPassword123!\n"), &stdout, &stderr)

	if err != nil {
		t.Fatalf("create-admin failed: %v, stderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Created admin user: admin@example.com") {
		t.Fatalf("unexpected create-admin output: %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"doctor", "--database-dialect", "sqlite", "--database-dsn", dbPath, "--capacity-profile", "compact"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("doctor failed: %v, stderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "profile=compact database=sqlite") {
		t.Fatalf("unexpected doctor output: %s", stdout.String())
	}
}

func TestRunReleaseVerify(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.bin"), []byte("bin"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := release.Build(root, release.Input{Edition: "site", CoreVersion: "1.0.0", SDKVersion: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signed, err := release.Sign(manifest, priv)
	if err != nil {
		t.Fatal(err)
	}
	signedBytes := []byte(fmt.Sprintf(`{"manifest":%s,"public_key":%q,"signature":%q}`, string(signed.Manifest), signed.PublicKey, signed.Signature))
	manifestFile := filepath.Join(root, "release.json")
	if err := os.WriteFile(manifestFile, signedBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	publicKeyFile := filepath.Join(root, "release.pub")
	if err := os.WriteFile(publicKeyFile, []byte(signed.PublicKey), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err = run([]string{"release", "verify", "--manifest", manifestFile, "--public-key", publicKeyFile}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("run release verify failed: %v, stderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "verified edition=site core=1.0.0") {
		t.Fatalf("unexpected release verify output: %s", stdout.String())
	}

	wrongPublic, _, _ := ed25519.GenerateKey(rand.Reader)
	wrongKeyFile := filepath.Join(root, "wrong.pub")
	if err := os.WriteFile(wrongKeyFile, []byte(base64.RawStdEncoding.EncodeToString(wrongPublic)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"release", "verify", "--manifest", manifestFile, "--public-key", wrongKeyFile}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("manifest verified with an unrelated trusted key")
	}
}

func TestRunBackupAndRestoreGates(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "hydro.sqlite")
	mediaDir := filepath.Join(root, "media")
	snapshot := filepath.Join(root, "snapshot")
	staging := filepath.Join(root, "staging")
	if err := os.MkdirAll(filepath.Join(mediaDir, "nested"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mediaDir, "nested", "asset.txt"), []byte("asset"), 0o640); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{"backup", "sqlite", "--database-dsn", dbPath, "--media-dir", mediaDir, "--destination", snapshot}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("backup failed: %v, stderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "backup-created") {
		t.Fatalf("unexpected backup output: %q", stdout.String())
	}

	stdout.Reset()
	if err := run([]string{"restore", "verify", "--snapshot", snapshot}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("restore verify failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "restore-verified") {
		t.Fatalf("unexpected restore verify output: %q", stdout.String())
	}

	if err := run([]string{"restore", "stage", "--snapshot", snapshot, "--destination", staging}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("restore stage succeeded without quarantine controls")
	}
	stdout.Reset()
	if err := run([]string{"restore", "stage", "--snapshot", snapshot, "--destination", staging, "--sessions-revoked", "--egress-quarantined", "--old-writes-blocked"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("restore stage failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(staging, "RESTORE_QUARANTINED.json")); err != nil {
		t.Fatalf("restore marker missing: %v", err)
	}
	manifest, err := backupstore.ReadManifest(filepath.Join(snapshot, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := backupstore.Verify(staging, manifest); err != nil {
		t.Fatalf("staged restore failed verification: %v", err)
	}
}

func TestRunPortableExportImportRequiresExplicitConfirmation(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "hydro.sqlite")
	documentPath := filepath.Join(root, "portable.json")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"migrate", "--database-dsn", dbPath}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	if err := run([]string{"portable", "export", "--database-dsn", dbPath, "--path", documentPath}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("portable export failed: %v", err)
	}
	if _, err := os.Stat(documentPath); err != nil {
		t.Fatalf("portable document missing: %v", err)
	}
	stdout.Reset()
	if err := run([]string{"portable", "import", "--path", documentPath, "--dry-run"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("portable dry-run failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "portable-import-valid") {
		t.Fatalf("unexpected dry-run output: %q", stdout.String())
	}
	if err := run([]string{"portable", "import", "--database-dsn", dbPath, "--path", documentPath}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("portable import succeeded without explicit confirmation")
	}
	if err := run([]string{"portable", "import", "--database-dsn", dbPath, "--path", documentPath, "--confirm"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("portable import failed: %v", err)
	}
}

func TestCheckDirectoryWritable(t *testing.T) {
	if err := checkDirectoryWritable(t.TempDir()); err != nil {
		t.Fatalf("writable directory rejected: %v", err)
	}
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := checkDirectoryWritable(file); err == nil {
		t.Fatal("file accepted as writable directory")
	}
}

func TestRunDoctorJSON(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "hydro.yaml")
	configData := fmt.Sprintf("profile: compact\ndatabase:\n  dialect: sqlite\n  dsn: %q\nmedia:\n  dir: %q\n", filepath.Join(root, "hydro.sqlite"), filepath.Join(root, "media"))
	if err := os.WriteFile(configPath, []byte(configData), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"migrate", "--config", configPath}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	stdout.Reset()
	if err := run([]string{"doctor", "--config", configPath, "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("doctor --json failed: %v", err)
	}
	var report struct {
		Version         string `json:"version"`
		SchemaMigration string `json:"schema_migration"`
		Database        string `json:"database"`
		MediaWritable   bool   `json:"media_writable"`
		BackupTools     string `json:"backup_tools"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Version != version || report.SchemaMigration == "" || report.Database != "sqlite" || !report.MediaWritable || report.BackupTools != "sqlite" {
		t.Fatalf("unexpected doctor report: %+v", report)
	}
}

func TestRunBackupPruneRequiresPreviewAndConfirmation(t *testing.T) {
	root := t.TempDir()
	for _, item := range []struct {
		name string
		when time.Time
	}{
		{name: "snapshot-old", when: time.Unix(1, 0)},
		{name: "snapshot-new", when: time.Unix(2, 0)},
	} {
		dir := filepath.Join(root, item.name)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := backupstore.WriteManifest(filepath.Join(dir, "manifest.json"), backupstore.Manifest{Version: 1, CreatedAt: item.when, Database: backupstore.FileDigest{Path: "database", SHA256: "0", Bytes: 0}}); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"backup", "prune", "--root", root, "--keep", "1", "--dry-run"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("backup prune dry-run failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "snapshot-old") {
		t.Fatalf("dry-run did not list old snapshot: %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, "snapshot-old")); err != nil {
		t.Fatalf("dry-run removed snapshot: %v", err)
	}
	if err := run([]string{"backup", "prune", "--root", root, "--keep", "1"}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("backup prune succeeded without preview or confirmation")
	}
	if err := run([]string{"backup", "prune", "--root", root, "--keep", "1", "--confirm"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("backup prune confirm failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "snapshot-old")); !os.IsNotExist(err) {
		t.Fatalf("old snapshot still exists, stat error = %v", err)
	}
}
