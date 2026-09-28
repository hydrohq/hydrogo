package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/testmysql"
	mysql "github.com/go-sql-driver/mysql"
)

func TestMySQLDumpRestoreContract(t *testing.T) {
	source := testmysql.Open(t)
	testmysql.MigrateAll(t, source)
	tools, err := LocateMySQLTools()
	if err != nil {
		t.Skip(err.Error())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	marker := fmt.Sprintf("mysql-restore-%d", time.Now().UnixNano())
	if _, err := source.ExecContext(ctx, "INSERT INTO pages (id,title,version,locale,slug,body,status,revision,publication_generation,created_at,updated_at) VALUES (?, ?, 1, 'en', ?, 'restore body', 'draft', 1, 0, NOW(), NOW())", marker, "restore marker", marker); err != nil {
		t.Fatal(err)
	}
	cfg, err := mysql.ParseDSN(os.Getenv("HYDRO_TEST_MYSQL_URL"))
	if err != nil {
		t.Fatal(err)
	}
	if err := source.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&cfg.DBName); err != nil {
		t.Fatal(err)
	}
	mediaDir := filepath.Join(t.TempDir(), "media")
	if err := os.MkdirAll(mediaDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mediaDir, "asset.bin"), []byte("restore media"), 0o640); err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(t.TempDir(), "snapshot")
	manifest, err := SnapshotMySQL(ctx, tools, cfg.FormatDSN(), mediaDir, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	adminCfg := *cfg
	adminCfg.DBName = "mysql"
	admin, err := database.Open(ctx, database.MySQL, adminCfg.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	targetDB := fmt.Sprintf("hydro_restore_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE `"+targetDB+"` CHARACTER SET utf8mb4"); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.ExecContext(context.Background(), "DROP DATABASE IF EXISTS `"+targetDB+"`")
		_ = admin.Close()
	})
	targetCfg := *cfg
	targetCfg.DBName = targetDB
	targetDSN := targetCfg.FormatDSN()
	staging := filepath.Join(t.TempDir(), "staging")
	controls := RestoreControls{SessionsRevoked: true, EgressQuarantined: true, OldWritesBlocked: true}
	if err := RestoreMySQLToStaging(ctx, tools, targetDSN, snapshot, staging, manifest, controls); err != nil {
		t.Fatalf("restore snapshot %s: %v", snapshot, err)
	}
	target, err := database.Open(ctx, database.MySQL, targetDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	var title string
	if err := target.QueryRowContext(ctx, "SELECT title FROM pages WHERE id = ?", marker).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "restore marker" {
		t.Fatalf("restored title = %q", title)
	}
	var triggerCount int
	if err := target.QueryRowContext(ctx, `SELECT COUNT(*) FROM information_schema.triggers WHERE trigger_schema = DATABASE() AND trigger_name IN ('audit_events_immutable_update','audit_events_immutable_delete')`).Scan(&triggerCount); err != nil {
		t.Fatal(err)
	}
	if triggerCount != 2 {
		t.Fatalf("restored immutable audit triggers = %d, want 2", triggerCount)
	}
	if err := ApplyRestoreControls(ctx, target, database.MySQL, controls); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(staging, "RESTORE_QUARANTINED.json")); err != nil {
		t.Fatal(err)
	}
	media, err := os.ReadFile(filepath.Join(staging, "media", "asset.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if string(media) != "restore media" {
		t.Fatalf("restored media = %q", media)
	}
}

func TestMySQLClientOptionFileProtectsCredentials(t *testing.T) {
	cfg := mysql.NewConfig()
	cfg.User, cfg.Passwd, cfg.Net, cfg.Addr = "backup-user", `p\"ass`, "tcp", "db.example:3306"
	path, err := mysqlClientOptionFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("option file permissions = %o", info.Mode().Perm())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{`user="backup-user"`, `password="p\\\"ass"`, `host="db.example"`, "port=3306"} {
		if !strings.Contains(string(data), value) {
			t.Fatalf("option file missing %q: %s", value, data)
		}
	}
}
