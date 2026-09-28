package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	adminstudiosql "github.com/ardaozyildirim/hydrogo/internal/adminstudio/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	mailstore "github.com/ardaozyildirim/hydrogo/internal/mailer/sqlstore"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestWeeklyDigestQueuesOneOptedInAdminMail(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "weekly.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	users := authstore.NewUserStore(db, database.SQLite)
	if err := users.Create(ctx, &auth.User{ID: "admin-1", Email: "admin@example.test", PasswordHash: "unused", Role: auth.RoleAdmin, Active: true}); err != nil {
		t.Fatal(err)
	}
	adminStore := adminstudiosql.Store{Dialect: database.SQLite}
	if err := adminStore.SetDigestPreference(ctx, db, "admin-1", "Europe/Istanbul", true); err != nil {
		t.Fatal(err)
	}
	tx := database.NewTenantTransactor(db, database.SQLite, false)
	spool := mailstore.Spool{DB: db, Tx: tx, Dialect: database.SQLite, MaxPending: 10}
	now := time.Date(2026, 9, 28, 6, 0, 0, 0, time.UTC)
	queued, err := runAdminWeeklyDigestOnce(ctx, now, db, tx, adminStore, users, spool, nil, false, false)
	if err != nil || queued != 1 {
		t.Fatalf("queued=%d err=%v", queued, err)
	}
	queued, err = runAdminWeeklyDigestOnce(ctx, now, db, tx, adminStore, users, spool, nil, false, false)
	if err != nil || queued != 0 {
		t.Fatalf("duplicate run queued=%d err=%v", queued, err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM mail_spool WHERE recipient=?", "admin@example.test").Scan(&count); err != nil || count != 1 {
		t.Fatalf("queued rows=%d err=%v", count, err)
	}
}
