package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	backupstore "github.com/ardaozyildirim/hydrogo/internal/backup"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/i18n"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	adminsearch "github.com/ardaozyildirim/hydrogo/internal/search"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/designsystem"
)

// adminStudioRoutes registers admin search, the studio manifest and the
// signed admin extension surfaces.
func (a *serveApp) adminStudioRoutes() []router.Route {
	cfg, db, modules, extensionRegistry := a.cfg, a.db, a.modules, a.extensionRegistry
	var routes []router.Route
	if modules["pages"] || modules["news"] || modules["members"] || modules["media"] || modules["site"] {
		routes = append(routes, router.Route{Pattern: "GET /admin/api/search", Access: router.Protected, Handler: adminsearch.AdminSearchHandler{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope, Modules: modules}})
	}
	routes = append(routes, router.Route{Pattern: "GET /admin/studio/manifest", Access: router.Protected, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if !activePlatformAdmin(user) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(adminstudio.E2EManifestForCollections(a.activeStudioCollections()))
	})})
	routes = append(routes, router.Route{Pattern: "GET /admin/extensions/manifest", Access: router.Protected, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if !activePlatformAdmin(user) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(extensionRegistry.Manifests())
	})})
	routes = append(routes, router.Route{Pattern: "GET /admin/extensions/{extension_id}/{surface_id}", Access: router.Protected, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		session := auth.SessionFromContext(r.Context())
		if user == nil || session == nil || !user.Active || user.Role != auth.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		principal := adminstudio.ChannelPrincipal{SessionID: session.ID, UserID: user.ID}
		if scope, scopeErr := tenant.FromContext(r.Context()); scopeErr == nil {
			principal.TenantID = scope.TenantID
		}
		extensionRegistry.ServeExtensionSurface(w, r, r.PathValue("extension_id"), r.PathValue("surface_id"), cfg.I18N.DefaultLocale, principal, func(p string) bool { return auth.Can(user, p) })
	})})
	routes = append(routes,
		router.Route{Pattern: "POST /admin/extensions/channels/bind", Access: router.Protected, Handler: adminExtensionChannelBindHandler(extensionRegistry)},
		router.Route{Pattern: "POST /admin/extensions/channels/revoke", Access: router.Protected, Handler: adminExtensionChannelRevokeHandler(extensionRegistry)},
	)
	return routes
}

// adminRoutes registers the admin dashboard, reports, audit explorer,
// backups, account locale, control-centre areas and IP bans.
func (a *serveApp) adminRoutes() []router.Route {
	var routes []router.Route
	adminDashboardHandler := http.HandlerFunc(a.adminDashboard)
	routes = append(routes, router.Route{Pattern: "GET /admin", Access: router.Protected, Handler: adminDashboardHandler})
	routes = append(routes, router.Route{Pattern: "GET /admin/reports", Access: router.Protected, Handler: adminDashboardHandler})
	if a.modules["pages"] {
		routes = append(routes, router.Route{Pattern: "GET /admin/audit", Access: router.Protected, Handler: http.HandlerFunc(a.adminAuditPage(auditstore.Store{DB: a.db, Dialect: a.cfg.Database.Dialect, RequireTenantScope: a.requireTenantScope}))})
	}
	routes = append(routes, adminBackupRoutes(http.HandlerFunc(a.adminBackupHandler(a.newRateLimiter(3, time.Minute))))...)
	routes = append(routes, router.Route{Pattern: "POST /admin/account/locale", Access: router.Protected, Handler: http.HandlerFunc(a.adminAccountLocale)})
	routes = append(routes, router.Route{Pattern: "GET /admin/reports/export", Access: router.Protected, Handler: http.HandlerFunc(a.adminReportsExport)})
	adminControlHandler := http.HandlerFunc(a.adminControl)
	for _, pattern := range adminControlRoutePatterns(a.modules) {
		routes = append(routes, router.Route{Pattern: pattern, Access: router.Protected, Handler: adminControlHandler})
	}
	routes = append(routes, adminSecurityBanRoutes(a.adminBanMutation(false), a.adminBanMutation(true))...)
	return routes
}

func (a *serveApp) adminAuditPage(auditExplorer auditstore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if !activePlatformAdmin(user) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		query := r.URL.Query()
		filter := auditstore.ExplorerFilter{Actor: query.Get("actor"), Action: query.Get("action"), Resource: query.Get("resource"), Target: query.Get("target"), Limit: 50}
		if raw := query.Get("offset"); raw != "" {
			var err error
			filter.Offset, err = strconv.Atoi(raw)
			if err != nil || filter.Offset < 0 || filter.Offset > 1_000_000 {
				http.Error(w, "invalid audit page", http.StatusBadRequest)
				return
			}
		}
		for _, value := range []string{filter.Actor, filter.Action, filter.Resource, filter.Target} {
			if len(value) > 256 {
				http.Error(w, "invalid audit filter", http.StatusBadRequest)
				return
			}
		}
		for _, pair := range []struct {
			input     string
			exclusive bool
		}{{"from", false}, {"until", true}} {
			if raw := query.Get(pair.input); raw != "" {
				day, err := time.Parse("2006-01-02", raw)
				if err != nil {
					http.Error(w, "invalid date filter", http.StatusBadRequest)
					return
				}
				if pair.exclusive {
					filter.Until = day.AddDate(0, 0, 1).UTC().Format("2006-01-02T15:04:05.000000000Z")
				} else {
					filter.From = day.UTC().Format("2006-01-02T15:04:05.000000000Z")
				}
			}
		}
		result, err := auditExplorer.Explore(r.Context(), filter)
		if err != nil {
			slog.Error("admin audit explorer unavailable", "error", err)
			http.Error(w, "Audit history is temporarily unavailable.", http.StatusInternalServerError)
			return
		}
		page := designsystem.AdminAuditPageProps{Actor: filter.Actor, Action: filter.Action, Resource: filter.Resource, Target: filter.Target, From: query.Get("from"), Until: query.Get("until"), Total: result.Total, Offset: filter.Offset, Limit: filter.Limit, Entries: make([]designsystem.AdminAuditEntry, 0, len(result.Events))}
		for _, event := range result.Events {
			at := event.CreatedAt
			legacy := at == "1970-01-01T00:00:00.000000000Z"
			if timestamp, parseErr := time.Parse("2006-01-02T15:04:05.000000000Z", at); parseErr == nil && !legacy {
				at = timestamp.Local().Format("02 Jan 2006 15:04:05 MST")
			} else if legacy {
				at = "Older event"
			}
			actor := event.ActorEmail
			if actor == "" {
				actor = "System"
				if event.ActorID != "" && !legacy {
					actor = "Unavailable user"
				}
			}
			targetLabel := ""
			if event.TargetID != "" {
				targetLabel = "Record"
			}
			page.Entries = append(page.Entries, designsystem.AdminAuditEntry{At: at, Actor: actor, Action: event.Action, Resource: event.ResourceType, Target: targetLabel, RequestID: event.RequestID, BeforeJSON: event.BeforeJSON, AfterJSON: event.AfterJSON, Legacy: legacy})
		}
		if err := ui.RenderAdminAudit(w, adminLocaleFromRequest(r), page); err != nil {
			slog.Error("admin audit page render failed", "error", err)
			http.Error(w, "Audit history is temporarily unavailable.", http.StatusInternalServerError)
		}
	}
}

// adminBackupHandler lists recent cloud snapshots (GET) and streams a signed
// on-demand backup after password confirmation (POST).
func (a *serveApp) adminBackupHandler(backupConfirmationLimiter security.RateLimiter) http.HandlerFunc {
	cfg, db, authSvc, auditRecorder := a.cfg, a.db, a.authSvc, a.auditRecorder
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if !activePlatformAdmin(user) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodGet {
			recent := make([]designsystem.AdminBackupItem, 0, 12)
			publicKey := ""
			if signingKey, err := backupstore.ParseSigningSeed(cfg.Backup.SigningSeed); err == nil {
				publicKey = hex.EncodeToString(signingKey[32:])
			}
			if cfg.Backup.Enabled {
				var snapshots []backupstore.SnapshotSummary
				dispatcher, err := backupstore.NewS3Dispatcher(backupstore.S3Config{Endpoint: cfg.Media.S3Endpoint, Bucket: cfg.Media.S3Bucket, Region: cfg.Media.S3Region, AccessKey: cfg.Media.S3AccessKey, SecretKey: cfg.Media.S3SecretKey, Prefix: cfg.Backup.Prefix, Secure: cfg.Media.S3UseSSL, ForcePathStyle: cfg.Media.S3ForcePathStyle})
				if err == nil {
					listCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
					snapshots, err = dispatcher.ListRecent(listCtx, 12)
					cancel()
				}
				if err != nil {
					slog.Error("admin backup history unavailable", "error", err)
				} else {
					for _, snapshot := range snapshots {
						recent = append(recent, designsystem.AdminBackupItem{Name: snapshot.Name, Created: snapshot.CreatedAt.Local().Format("02 Jan 2006 15:04 MST"), Size: fmt.Sprintf("%.1f MB", float64(snapshot.Bytes)/(1024*1024))})
					}
				}
			}
			if err := ui.RenderAdminBackup(w, security.CSRFTokenFromContext(r.Context()), adminLocaleFromRequest(r), recent, cfg.Backup.Enabled, publicKey); err != nil {
				http.Error(w, "backup page unavailable", http.StatusInternalServerError)
			}
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		if !security.AllowForRequest(backupConfirmationLimiter, r.Context(), "admin-backup:"+user.ID) {
			http.Error(w, "Please wait before trying again.", http.StatusTooManyRequests)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid backup request", http.StatusBadRequest)
			return
		}
		signingKey, err := backupstore.ParseSigningSeed(cfg.Backup.SigningSeed)
		if err != nil {
			http.Error(w, "Backup signing is not configured.", http.StatusServiceUnavailable)
			return
		}
		confirmed, err := authSvc.Authenticate(r.Context(), user.Email, r.FormValue("password"))
		if err != nil || confirmed.ID != user.ID || confirmed.Role != auth.RoleAdmin {
			http.Error(w, "Password confirmation failed.", http.StatusForbidden)
			return
		}
		workDir, err := os.MkdirTemp("", "hydrogo-admin-backup-")
		if err != nil {
			http.Error(w, "Backup is temporarily unavailable.", http.StatusInternalServerError)
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
			http.Error(w, "Backup could not be created.", http.StatusInternalServerError)
			return
		}
		if auditRecorder != nil {
			if err := auditRecorder.Record(r.Context(), "backup.downloaded", user.ID); err != nil {
				slog.Error("admin backup audit failed", "error", err)
				http.Error(w, "Backup could not be created.", http.StatusInternalServerError)
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

func (a *serveApp) adminAccountLocale(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	locale := strings.TrimSpace(r.FormValue("locale"))
	if !i18n.IsSupportedCatalogLocale(locale) {
		http.Error(w, "unsupported locale", http.StatusBadRequest)
		return
	}
	setAdminLocaleCookie(w, locale, a.cfg.Security.SecureCookies)
	http.Redirect(w, r, "/admin/account?locale_saved=1", http.StatusSeeOther)
}

// adminControl renders the control centre and its areas (site, security,
// operations, account, users, permissions, developer tools).
func (a *serveApp) adminControl(w http.ResponseWriter, r *http.Request) {
	cfg, db, modules := a.cfg, a.db, a.modules
	userStore, authSvc, mfaService, recoveryService, banStore := a.userStore, a.authSvc, a.mfaService, a.recoveryService, a.banStore
	user := auth.UserFromContext(r.Context())
	if !adminControlAllowed(user, r.URL.Path) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	active := "control"
	if r.URL.Path == "/admin/site" {
		active = "site"
	}
	if r.URL.Path == "/admin/media" {
		active = "media"
	}
	if r.URL.Path == "/admin/security" {
		active = "security"
	}
	if r.URL.Path == "/admin/operations" {
		active = "operations"
	}
	if r.URL.Path == "/admin/developer" {
		if err := ui.RenderAdminArea(w, ui.AdminAreaData{Request: r, Email: user.Email, CSRF: security.CSRFTokenFromContext(r.Context()), Area: "developer", Locale: adminLocaleFromRequest(r)}); err != nil {
			http.Error(w, "developer tools unavailable", http.StatusInternalServerError)
		}
		return
	}
	if r.URL.Path == "/admin/developer/resources" {
		if err := ui.RenderAdminResourceUsage(w, ui.AdminResourceUsageData{Email: user.Email, Snapshot: a.metrics.Snapshot(), Locale: adminLocaleFromRequest(r)}); err != nil {
			http.Error(w, "resource usage unavailable", http.StatusInternalServerError)
		}
		return
	}
	if r.URL.Path == "/admin/developer/components" {
		http.Redirect(w, r, "/admin/developer", http.StatusMovedPermanently)
		return
	}
	if r.URL.Path == "/admin/account" {
		active = "account"
	}
	if r.URL.Path == "/admin/users" {
		active = "users"
	}
	if r.URL.Path == "/admin/permissions" {
		active = "permissions"
	}
	var err error
	if active == "control" {
		err = ui.RenderAdminControl(w, ui.AdminControlData{Email: user.Email, CSRF: security.CSRFTokenFromContext(r.Context()), Active: active, Locale: adminLocaleFromRequest(r)})
	} else {
		scope, _ := tenant.FromContext(r.Context())
		areaData := ui.AdminAreaData{Request: r, Email: user.Email, CSRF: security.CSRFTokenFromContext(r.Context()), Area: active, Locale: adminLocaleFromRequest(r), TenantID: scope.TenantID, LocaleSaved: r.URL.Query().Get("locale_saved") == "1", SoundAlertsEnabled: modules["realtime"] && modules["forms"], FontToolsEnabled: modules["pages"], EmailVerificationEnabled: a.smtpSender != nil, PasswordResetLinkEnabled: recoveryService != nil && cfg.HTTP.PublicOrigin != "", Permissions: authSvc.AvailablePermissions()}
		if active == "account" {
			areaData.MFAEnabled, err = mfaService.Enabled(r.Context(), db, user.ID)
			areaData.MFAStatusLoaded = err == nil
			keySize := len([]byte(cfg.Security.MFAMasterKey))
			areaData.MFAAvailable = areaData.MFAStatusLoaded && (keySize == 16 || keySize == 24 || keySize == 32)
			err = nil // Keep account settings available if the MFA status store is temporarily unavailable.
		}
		if active == "users" {
			if repo, ok := any(userStore).(auth.UserAdminRepository); ok {
				areaData.Users, err = repo.List(r.Context())
			} else {
				err = fmt.Errorf("admin user repository unavailable")
			}
		}
		if active == "security" {
			entries, listErr := banStore.ListActive(r.Context(), db, time.Now().UTC(), 4096)
			if listErr != nil {
				err = listErr
			} else {
				areaData.IPBans = make([]ui.SecurityBanRow, 0, len(entries))
				for _, entry := range entries {
					expires := "Süresiz"
					if !entry.Expires.IsZero() {
						expires = entry.Expires.UTC().Format(time.RFC3339)
					}
					areaData.IPBans = append(areaData.IPBans, ui.SecurityBanRow{Address: entry.Prefix.String(), Expires: expires})
				}
			}
		}
		if err == nil {
			if active == "account" {
				var page bytes.Buffer
				err = ui.RenderAdminArea(&page, areaData)
				if err == nil {
					w.Header().Set("Cache-Control", "private, no-store")
					w.Header().Set("X-Hydro-Custom-CSP", security.InlineCSPDirectives(page.Bytes()))
					_, err = w.Write(page.Bytes())
				}
			} else {
				err = ui.RenderAdminArea(w, areaData)
			}
		}
	}
	if err != nil {
		http.Error(w, "admin control center unavailable", http.StatusInternalServerError)
	}
}

// adminBanMutation adds or removes an IP ban rule and updates the in-memory
// blocklist.
func (a *serveApp) adminBanMutation(remove bool) http.HandlerFunc {
	db, banStore, ipBlocklist, authHTTP := a.db, a.banStore, a.ipBlocklist, a.authHTTP
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if !activePlatformAdmin(user) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid IP rule", http.StatusBadRequest)
			return
		}
		prefix, err := parseIPBanPrefix(r.FormValue("address"))
		if err != nil {
			http.Error(w, "invalid IP address or CIDR", http.StatusBadRequest)
			return
		}
		if remove {
			err = banStore.Remove(r.Context(), db, prefix)
			if err == nil {
				err = ipBlocklist.Unban(prefix.String())
			}
		} else {
			var expires time.Time
			rawDuration := strings.TrimSpace(r.FormValue("duration"))
			if rawDuration != "" {
				duration, parseErr := time.ParseDuration(rawDuration)
				if parseErr != nil || duration < 0 || duration > 365*24*time.Hour {
					http.Error(w, "invalid ban duration", http.StatusBadRequest)
					return
				}
				if duration > 0 {
					expires = time.Now().UTC().Add(duration)
				}
			}
			err = banStore.Put(r.Context(), db, prefix, expires)
			if err == nil {
				err = ipBlocklist.Ban(prefix.String(), expires)
			}
		}
		if err != nil {
			http.Error(w, "IP rule could not be updated", http.StatusServiceUnavailable)
			return
		}
		action := "security.ip_ban.created"
		if remove {
			action = "security.ip_ban.removed"
		}
		if authHTTP.Audit != nil {
			_ = authHTTP.Audit.Record(r.Context(), action, prefix.String())
		}
		http.Redirect(w, r, "/admin/security", http.StatusSeeOther)
	}
}

func adminControlRoutePatterns(enabled map[string]bool) []string {
	patterns := []string{
		"GET /admin/control",
		"GET /admin/security",
		"GET /admin/operations",
		"GET /admin/account",
		"GET /admin/users",
		"GET /admin/permissions",
		"GET /admin/developer",
		"GET /admin/developer/resources",
	}
	if enabled["site"] {
		patterns = append(patterns, "GET /admin/site")
	}
	return patterns
}

func adminControlAllowed(user *auth.User, path string) bool {
	if path == "/admin/account" {
		return user != nil && user.Active
	}
	if path == "/admin/site" {
		return auth.Can(user, auth.PermissionSiteManage)
	}
	if !activePlatformAdmin(user) {
		return false
	}
	switch path {
	case "/admin/users", "/admin/permissions":
		return auth.Can(user, auth.PermissionUsersManage)
	case "/admin/operations":
		return auth.Can(user, auth.PermissionOperations)
	}
	return true
}
