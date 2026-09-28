package sqlstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func TestListByUserIDReturnsOnlyActiveSessionsWithoutSecretProjection(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "sessions.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE sessions(id TEXT PRIMARY KEY,user_id TEXT,token_hash TEXT,expires_at TEXT,created_at TEXT,last_seen_at TEXT,user_agent TEXT,ip_address TEXT,tenant_id TEXT)`)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, row := range [][]any{{"current", "u", "hash-current", now.Add(time.Hour).Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), "Browser", "127.0.0.1", ""}, {"expired", "u", "hash-old", now.Add(-time.Hour).Format(time.RFC3339Nano), now.Add(-2 * time.Hour).Format(time.RFC3339Nano), now.Add(-2 * time.Hour).Format(time.RFC3339Nano), "Old", "127.0.0.2", ""}, {"other", "other", "hash-other", now.Add(time.Hour).Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), "Other", "127.0.0.3", ""}} {
		_, err = db.Exec(`INSERT INTO sessions VALUES(?,?,?,?,?,?,?,?,?)`, row...)
		if err != nil {
			t.Fatal(err)
		}
	}
	store := NewSessionStore(db, database.SQLite)
	items, err := store.ListByUserID(context.Background(), "u", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "current" || items[0].TokenHash != "hash-current" {
		t.Fatalf("sessions=%+v", items)
	}
}
