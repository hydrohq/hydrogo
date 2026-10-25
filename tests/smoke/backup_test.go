//go:build smoke

package smoke

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBackupRestoreKeepsData takes a snapshot of a running SQLite project,
// verifies and stages it the way the restore runbook does, and checks that a
// server started on the restored files serves the same records and accepts
// the same administrator. The restore tooling for PostgreSQL has its own
// tests against a real database in internal/backup.
func TestBackupRestoreKeepsData(t *testing.T) {
	if d := os.Getenv("SMOKE_DIALECT"); d != "" && d != "sqlite" {
		t.Skip("snapshot restore of the file-based database")
	}
	p := start(t, binary, map[string]string{"notes": notesType})
	token, _ := p.serviceAccount("writer", "notes.read", "notes.write", "notes.all")
	for i := 0; i < 5; i++ {
		if res := p.must("POST", "/api/v1/admin/content/notes", &opts{token: token, body: map[string]any{"title": fmt.Sprintf("note %d", i)}}); res.status != 201 {
			t.Fatalf("create: %d %s", res.status, res.body)
		}
	}

	snapshot := filepath.Join(t.TempDir(), "snapshot")
	p.run("", "backup", "sqlite", "--destination", snapshot, "--config", p.config)
	p.run("", "restore", "verify", "--snapshot", snapshot)
	staged := filepath.Join(t.TempDir(), "staged")
	p.run("", "restore", "stage", "--snapshot", snapshot, "--destination", staged, "--sessions-revoked", "--egress-quarantined", "--old-writes-blocked")

	// Restore into a second server: same project, the staged database.
	raw, err := os.ReadFile(p.config)
	if err != nil {
		t.Fatal(err)
	}
	port := freePort(t)
	cfg := strings.NewReplacer(
		fmt.Sprintf("127.0.0.1:%d", p.port), fmt.Sprintf("127.0.0.1:%d", port),
		`dsn: "data/hydro.db"`, fmt.Sprintf("dsn: %q", filepath.Join(staged, "database")),
	).Replace(string(raw))
	restored := filepath.Join(p.dir, "restored.yaml")
	if err := os.WriteFile(restored, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	q := p.serveOn(restored, port)
	list := q.must("GET", "/api/v1/admin/content/notes?limit=100", &opts{token: token}).list()
	if len(list) != 5 {
		t.Errorf("the restored server has %d notes, want 5", len(list))
	}
	if _, res := q.login("admin@smoke.test", "Quartz-Meadow-4417-Lantern"); res.status >= 300 {
		t.Errorf("the administrator cannot sign in after the restore: %d %s", res.status, res.body)
	}
}
