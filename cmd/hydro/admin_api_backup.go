package main

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	backupstore "github.com/ardaozyildirim/hydrogo/internal/backup"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

type adminBackupItem struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Bytes     int64     `json:"bytes"`
}

type adminBackupStatus struct {
	Enabled   bool              `json:"enabled"`
	PublicKey string            `json:"public_key,omitempty"`
	Recent    []adminBackupItem `json:"recent"`
}

// adminBackupStatusHandler lists recent cloud snapshots and the signing key.
func (a *serveApp) adminBackupStatusHandler(w http.ResponseWriter, r *http.Request) {
	if !activePlatformAdmin(auth.UserFromContext(r.Context())) {
		api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
		return
	}
	cfg := a.cfg
	status := adminBackupStatus{Enabled: cfg.Backup.Enabled, Recent: []adminBackupItem{}}
	if signingKey, err := backupstore.ParseSigningSeed(cfg.Backup.SigningSeed); err == nil {
		status.PublicKey = hex.EncodeToString(signingKey[32:])
	}
	if cfg.Backup.Enabled {
		dispatcher, err := backupstore.NewS3Dispatcher(backupstore.S3Config{Endpoint: cfg.Media.S3Endpoint, Bucket: cfg.Media.S3Bucket, Region: cfg.Media.S3Region, AccessKey: cfg.Media.S3AccessKey, SecretKey: cfg.Media.S3SecretKey, Prefix: cfg.Backup.Prefix, Secure: cfg.Media.S3UseSSL, ForcePathStyle: cfg.Media.S3ForcePathStyle})
		var snapshots []backupstore.SnapshotSummary
		if err == nil {
			listCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			snapshots, err = dispatcher.ListRecent(listCtx, 12)
			cancel()
		}
		if err != nil {
			slog.Error("admin backup history unavailable", "error", err)
		}
		for _, snapshot := range snapshots {
			status.Recent = append(status.Recent, adminBackupItem{Name: snapshot.Name, CreatedAt: snapshot.CreatedAt, Bytes: snapshot.Bytes})
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	api.WriteData(w, http.StatusOK, status, nil)
}

// adminBackupDownloadHandler streams a signed on-demand snapshot after the
// admin confirms their password.
func (a *serveApp) adminBackupDownloadHandler(limiter security.RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, db, authSvc, auditRecorder := a.cfg, a.db, a.authSvc, a.auditRecorder
		user := auth.UserFromContext(r.Context())
		if !activePlatformAdmin(user) {
			api.WriteError(w, r, http.StatusForbidden, "forbidden", nil)
			return
		}
		if !security.AllowForRequest(limiter, r.Context(), "admin-backup:"+user.ID) {
			api.WriteError(w, r, http.StatusTooManyRequests, "please wait before trying again", nil)
			return
		}
		var input struct {
			Password string `json:"password"`
		}
		if !api.DecodeJSON(w, r, &input, 8<<10) {
			return
		}
		signingKey, err := backupstore.ParseSigningSeed(cfg.Backup.SigningSeed)
		if err != nil {
			api.WriteError(w, r, http.StatusServiceUnavailable, "backup signing is not configured", nil)
			return
		}
		confirmed, err := authSvc.Authenticate(r.Context(), user.Email, input.Password)
		if err != nil || confirmed.ID != user.ID || confirmed.Role != auth.RoleAdmin {
			api.WriteError(w, r, http.StatusForbidden, "password confirmation failed", map[string]string{"password": "invalid"})
			return
		}
		workDir, err := os.MkdirTemp("", "hydrogo-admin-backup-")
		if err != nil {
			api.WriteError(w, r, http.StatusInternalServerError, "backup is temporarily unavailable", nil)
			return
		}
		defer os.RemoveAll(workDir)
		snapshotDir := filepath.Join(workDir, "snapshot")
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
		defer cancel()
		switch cfg.Database.Dialect {
		case database.SQLite:
			_, err = backupstore.SnapshotSQLite(ctx, db, cfg.Media.Dir, snapshotDir)
		case database.Postgres:
			var tools backupstore.PostgresTools
			tools, err = backupstore.LocatePostgresTools()
			if err == nil {
				_, err = backupstore.SnapshotPostgres(ctx, tools, cfg.Database.DSN, cfg.Media.Dir, snapshotDir)
			}
		case database.MySQL:
			var tools backupstore.MySQLTools
			tools, err = backupstore.LocateMySQLTools()
			if err == nil {
				_, err = backupstore.SnapshotMySQL(ctx, tools, cfg.Database.DSN, cfg.Media.Dir, snapshotDir)
			}
		default:
			err = errors.New("unsupported backup database")
		}
		if err != nil {
			slog.Error("admin backup snapshot failed", "error", err)
			api.WriteError(w, r, http.StatusInternalServerError, "backup could not be created", nil)
			return
		}
		if auditRecorder != nil {
			if err := auditRecorder.Record(r.Context(), "backup.downloaded", user.ID); err != nil {
				slog.Error("admin backup audit failed", "error", err)
				api.WriteError(w, r, http.StatusInternalServerError, "backup could not be created", nil)
				return
			}
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Disposition", `attachment; filename="hydrogo-backup-`+time.Now().UTC().Format("20060102T150405Z")+`.tar.gz"`)
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if err := backupstore.WriteSignedSnapshotArchive(ctx, w, snapshotDir, signingKey); err != nil {
			slog.Error("admin backup archive stream failed", "error", err)
		}
	}
}
