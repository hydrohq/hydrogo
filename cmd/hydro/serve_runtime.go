package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	aicore "github.com/ardaozyildirim/hydrogo/internal/ai"
	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	analyticsstore "github.com/ardaozyildirim/hydrogo/internal/analytics/sqlstore"
	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	backupstore "github.com/ardaozyildirim/hydrogo/internal/backup"
	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/mailer"
	mailstore "github.com/ardaozyildirim/hydrogo/internal/mailer/sqlstore"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	pushservice "github.com/ardaozyildirim/hydrogo/internal/push"
	pushstore "github.com/ardaozyildirim/hydrogo/internal/push/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/realtime"
	"github.com/ardaozyildirim/hydrogo/internal/richtext"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	securitystore "github.com/ardaozyildirim/hydrogo/internal/security/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
	commercestore "github.com/ardaozyildirim/hydrogo/modules/commerce/repository/sqlstore"
	"github.com/ardaozyildirim/hydrogo/modules/members"
	memberstore "github.com/ardaozyildirim/hydrogo/modules/members/sqlstore"
	membertransport "github.com/ardaozyildirim/hydrogo/modules/members/transport"
	pagesstore "github.com/ardaozyildirim/hydrogo/modules/pages/repository/sqlstore"
	pageservice "github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

// openDatabase opens the database for the capacity profile, recovers SQLite,
// and loads the IP blocklist.
func (a *serveApp) openDatabase() error {
	ctx, cfg, stderr := a.ctx, a.cfg, a.stderr
	budget, err := capacity.Defaults(cfg.Profile)
	if err != nil {
		return err
	}
	a.budget = budget
	db, err := database.OpenWithOptions(ctx, cfg.Database.Dialect, cfg.Database.DSN, profileDatabaseOptions(budget, cfg.Profile))
	if err != nil {
		return err
	}
	a.db = db
	a.onClose(func() error {
		var closeErr error
		a.stop()
		if cfg.Database.Dialect == database.SQLite && !a.opts.cgi {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), budget.ShutdownTimeout)
			defer cancel()
			if err := database.CheckpointSQLite(shutdownCtx, db); err != nil {
				closeErr = errors.Join(closeErr, fmt.Errorf("SQLite shutdown checkpoint: %w", err))
			}
		}
		if err := db.Close(); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("close database: %w", err))
		}
		return closeErr
	})
	if cfg.Database.Dialect == database.SQLite {
		if err := database.RecoverSQLite(ctx, db); err != nil {
			return fmt.Errorf("SQLite startup recovery failed: %w", err)
		}
		if stderr != nil {
			_, _ = fmt.Fprintln(stderr, `{"event":"sqlite_recovery","integrity":"ok","wal_checkpoint":"restart"}`)
		}
	}
	if a.opts.cgi || a.opts.fastCGI {
		if err := database.ApplyMigrationsForModules(ctx, db, cfg.Database.Dialect, migrations.FS, a.modules); err != nil {
			return fmt.Errorf("auto-migration failed: %w", err)
		}
	}
	if !a.opts.cgi && !a.opts.fastCGI {
		// Without this, an unmigrated database fails on whichever table is read first
		// ("no such table: security_ip_bans"), which reads like a bug in HydroGo.
		if err := database.CheckCompatibilityForModules(ctx, db, cfg.Database.Dialect, migrations.FS, a.modules); err != nil {
			return fmt.Errorf("the database is not ready for this version (%w); run `hydro migrate --config <your config file>` first", err)
		}
	}
	a.transactioner = database.NewTenantTransactor(db, cfg.Database.Dialect, a.requireTenantScope)
	ipBlocklist := security.NewIPBlocklist(4096)
	banStore := securitystore.BanStore{Dialect: cfg.Database.Dialect}
	if err := refreshIPBlocklist(ctx, db, banStore, ipBlocklist); err != nil {
		return fmt.Errorf("load IP security rules: %w", err)
	}
	if !a.opts.cgi {
		a.spawn(func() { runIPBlocklistRefresh(ctx, db, banStore, ipBlocklist) })
	}
	a.ipBlocklist, a.banStore = ipBlocklist, banStore
	return nil
}

// initAuth wires admin accounts, sessions, MFA, recovery, invitations, Redis,
// the audit recorder and the member stores.
func (a *serveApp) initAuth() error {
	ctx, cfg, db, stderr := a.ctx, a.cfg, a.db, a.stderr
	requireTenantScope, transactioner := a.requireTenantScope, a.transactioner
	userStore := authstore.NewUserStore(db, cfg.Database.Dialect)
	sessionStore := authstore.NewSessionStore(db, cfg.Database.Dialect)
	userStore.RequireTenantScope = requireTenantScope
	sessionStore.RequireTenantScope = requireTenantScope
	mobileAuth := auth.NewMobileAuthService(db, cfg.Database.Dialect)
	authSvc := auth.NewAuthService(userStore, sessionStore, mobileAuth, auth.DefaultArgon2Params)
	authSvc.SetPermissionStore(userStore)
	apiTokenStore := authstore.NewAPITokenStore(db, cfg.Database.Dialect)
	apiTokenStore.RequireTenantScope = requireTenantScope
	authSvc.SetAPITokenStore(apiTokenStore)
	mfaService := &auth.MFAService{Store: userStore, MasterKey: []byte(cfg.Security.MFAMasterKey)}
	recoveryService := &auth.PasswordRecovery{Users: userStore, Tokens: userStore, Sessions: sessionStore, ArgonParams: auth.DefaultArgon2Params}
	invitationService := &auth.InvitationService{Users: userStore, Invites: userStore, ArgonParams: auth.DefaultArgon2Params}
	redisClient, err := newRedisRuntime(ctx, cfg.Redis.URL)
	if err != nil {
		if a.devMode || strings.EqualFold(string(cfg.Profile), "compact") {
			if stderr != nil {
				_, _ = fmt.Fprintf(stderr, "warning: Redis connection failed (%v); falling back to in-memory rate limiting and cache\n", err)
			}
			redisClient = redisRuntime{}
		} else {
			return fmt.Errorf("connect redis: %w", err)
		}
	}
	a.redisClient = redisClient
	a.onClose(func() error { _ = redisClient.Close(); return nil })
	newRateLimiter := a.newRateLimiter
	auditRecorder := newAuditRecorder(db, cfg.Database.Dialect, requireTenantScope)
	var socialOAuth *auth.SocialOAuthHandler
	if !cfg.SafeMode {
		socialOAuth, err = configuredSocialOAuth(db, cfg.Database.Dialect, authSvc, auditRecorder, cfg.Security.SecureCookies, requireTenantScope)
		if err != nil {
			return fmt.Errorf("configure social sign-in: %w", err)
		}
	}
	authHTTP := auth.HTTPHandler{Service: authSvc, Audit: auditRecorder, Secure: cfg.Security.SecureCookies, MFA: mfaService, MFARequiredRoles: configuredMFARoles(cfg.Security.MFARequiredRoles), MFADB: db, Transactions: transactioner, LoginLimiter: newRateLimiter(5, time.Minute), MFALimiter: newRateLimiter(5, time.Minute), RecoveryLimiter: newRateLimiter(3, time.Hour), RecoveryConfirmLimiter: newRateLimiter(10, time.Hour), InviteAcceptLimiter: newRateLimiter(10, time.Hour), LoginTarpit: security.NewTarpit(10000, 15*time.Minute, time.Second, 8*time.Second), Recovery: recoveryService, RecoveryDB: db, Invitation: invitationService, InviteDB: db, PublicOrigin: cfg.HTTP.PublicOrigin, SocialOAuth: socialOAuth, LoginSecurity: userStore, TokenLifetimeSettings: authstore.TokenLifetimeStore{DB: db, Dialect: cfg.Database.Dialect}, Logger: a.logger}
	authHTTP.PasswordResetTTL = cfg.PasswordResetDuration()
	authHTTP.InvitationTTL = cfg.InvitationDuration()
	mobileAuthHTTP := auth.MobileHTTPHandler{Auth: authSvc, Mobile: mobileAuth, MFA: mfaService, MFADB: db, LoginLimiter: newRateLimiter(5, time.Minute), MFALimiter: newRateLimiter(5, time.Minute), Audit: newAuditRecorder(db, cfg.Database.Dialect, requireTenantScope)}
	a.memberStore = memberstore.Store{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	a.userStore, a.sessionStore, a.authSvc, a.mfaService, a.recoveryService = userStore, sessionStore, authSvc, mfaService, recoveryService
	a.userAdmin = userStore
	a.roleStore = authstore.RoleStore{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	a.auditRecorder, a.authHTTP, a.mobileAuthHTTP = auditRecorder, authHTTP, mobileAuthHTTP
	return nil
}

// initOptionalModules wires AI assistance, push, realtime, edit presence,
// analytics queues and the member service when their modules are enabled.
func (a *serveApp) initOptionalModules() error {
	cfg, db, modules := a.cfg, a.db, a.modules
	requireTenantScope := a.requireTenantScope
	newRateLimiter := a.newRateLimiter
	memberSessionStore := memberstore.SessionStore{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	memberAuditStore := memberstore.AuditStore{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	if modules["ai"] && strings.EqualFold(strings.TrimSpace(os.Getenv("HYDRO_AI_ENABLED")), "true") {
		endpoint, apiKey, model := strings.TrimSpace(os.Getenv("HYDRO_AI_ENDPOINT")), os.Getenv("HYDRO_AI_API_KEY"), strings.TrimSpace(os.Getenv("HYDRO_AI_MODEL"))
		if endpoint == "" || strings.TrimSpace(apiKey) == "" || model == "" {
			return errors.New("AI assistance requires HYDRO_AI_ENDPOINT, HYDRO_AI_API_KEY, and HYDRO_AI_MODEL")
		}
		provider := aicore.OpenAICompatible{Endpoint: endpoint, APIKey: apiKey, Model: model}
		if provider.Validate() != nil {
			return errors.New("AI assistance provider configuration is invalid")
		}
		a.contentAssistant = aicore.Assistant{Enabled: true, Provider: provider, Timeout: 15 * time.Second}
		a.contentAIAudit = auditstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
	}
	var pushDispatcher *pushservice.Dispatcher
	if modules["push"] && !cfg.SafeMode {
		repository := pushstore.Store{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}
		dispatcher := &pushservice.Dispatcher{Repository: repository}
		vapidPublic, vapidPrivate, vapidSubject := os.Getenv("HYDRO_PUSH_VAPID_PUBLIC_KEY"), os.Getenv("HYDRO_PUSH_VAPID_PRIVATE_KEY"), os.Getenv("HYDRO_PUSH_VAPID_SUBJECT")
		if vapidPublic != "" || vapidPrivate != "" || vapidSubject != "" {
			if vapidPublic == "" || vapidPrivate == "" || vapidSubject == "" {
				return fmt.Errorf("HYDRO_PUSH_VAPID_PUBLIC_KEY, HYDRO_PUSH_VAPID_PRIVATE_KEY, and HYDRO_PUSH_VAPID_SUBJECT must be set together")
			}
			dispatcher.Web = pushservice.WebPushSender{Public: vapidPublic, Private: vapidPrivate, Subject: vapidSubject}
		}
		if credentialsPath := os.Getenv("HYDRO_PUSH_FCM_CREDENTIALS"); credentialsPath != "" {
			credentials, readErr := os.ReadFile(credentialsPath) // #nosec G304 ,G703-- operator-configured credentials path (HYDRO_PUSH_FCM_CREDENTIALS)
			if readErr != nil {
				return fmt.Errorf("read FCM credentials: %w", readErr)
			}
			fcm, fcmErr := pushservice.NewFCMSender(credentials, nil)
			if fcmErr != nil {
				return fmt.Errorf("configure FCM sender: %w", fcmErr)
			}
			dispatcher.Mobile = fcm
		}
		if dispatcher.Web == nil && dispatcher.Mobile == nil {
			return fmt.Errorf("the push module requires VAPID settings or HYDRO_PUSH_FCM_CREDENTIALS")
		}
		pushDispatcher = dispatcher
	}
	a.pushDispatcher = pushDispatcher
	if modules["realtime"] {
		a.realtimeHub = &realtime.Hub{RequireTenantScope: requireTenantScope}
		if err := configureRealtimeHub(a.realtimeHub, a.redisClient, cfg.Redis.KeyPrefix); err != nil {
			return fmt.Errorf("configure realtime hub: %w", err)
		}
		a.onClose(func() error { return a.realtimeHub.Close() })
	}
	if modules["pages"] || modules["news"] {
		a.editPresence = &realtime.Presence{TTL: realtime.DefaultPresenceTTL}
	}
	if modules["analytics"] {
		analyticsStore := analyticsstore.Store{Executor: db, Dialect: cfg.Database.Dialect}
		a.analyticsStore = analyticsStore
		contentViewQueue := analytics.NewContentViewQueue(analyticsStore, 256, 1)
		a.contentViewQueue = contentViewQueue
		a.onClose(func() error { contentViewQueue.Close(); return nil })
		if modules["search"] {
			searchQueryQueue := analytics.NewSearchQueryQueue(analyticsStore, 256, 1)
			a.searchQueryQueue = searchQueryQueue
			a.onClose(func() error { searchQueryQueue.Close(); return nil })
		}
		if cfg.Analytics.NoCookieEnabled {
			noCookieHasher, err := analytics.NewVisitorHasher([]byte(cfg.Analytics.HashKey))
			if err != nil {
				return fmt.Errorf("initialize local analytics: %w", err)
			}
			a.noCookieHasher = noCookieHasher
			noCookieQueue := analytics.NewNoCookieQueue(analyticsStore, 256)
			a.noCookieQueue = noCookieQueue
			a.onClose(func() error { noCookieQueue.Close(); return nil })
		}
	}
	if modules["members"] {
		a.memberService = members.Service{Members: a.memberStore, Sessions: memberSessionStore, Audit: memberAuditStore, ArgonParams: auth.DefaultArgon2Params}
		a.memberSessions = memberSessionStore
	}
	if modules["members"] {
		a.memberHTTP = membertransport.Handler{Service: a.memberService, Mobile: &members.MobileAuthService{DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}, Secure: cfg.Security.SecureCookies, Limiter: newRateLimiter(10, time.Minute), Audit: newAuditRecorder(db, cfg.Database.Dialect, requireTenantScope), DB: db, DefaultLocale: cfg.I18N.DefaultLocale, Push: pushDispatcher}
		if modules["commerce"] {
			a.memberHTTP.Orders = commerce.MemberOrdersReader{Repository: commercestore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope}}
		}
		if !cfg.SafeMode {
			memberOAuth, err := configuredMemberSocialOAuth(db, cfg.Database.Dialect, a.memberService, newAuditRecorder(db, cfg.Database.Dialect, requireTenantScope), cfg.Security.SecureCookies, requireTenantScope)
			if err != nil {
				return fmt.Errorf("configure member social sign-in: %w", err)
			}
			a.memberOAuthHTTP = memberOAuth
		}
	}
	return nil
}

// initMail configures outgoing email (SMTP or direct MX delivery) and member
// email verification.
func (a *serveApp) initMail() error {
	cfg, db, modules := a.cfg, a.db, a.modules
	var smtpSender mailer.Sender
	var emailBrandingSettings mailer.EmailBrandingProvider
	if modules["auth"] {
		emailBrandingSettings = mailstore.EmailBrandingStore{DB: db, Dialect: cfg.Database.Dialect}
	}
	decorateEmailSender := func(sender mailer.Sender) mailer.Sender {
		if emailBrandingSettings == nil {
			return sender
		}
		return mailer.BrandedSender{Next: sender, Settings: emailBrandingSettings, PublicOrigin: cfg.HTTP.PublicOrigin}
	}
	if cfg.Mail.Host != "" && !cfg.SafeMode {
		sender, err := mailer.NewSMTP(mailer.SMTPConfig{Host: cfg.Mail.Host, Port: cfg.Mail.Port, From: cfg.Mail.From, Username: cfg.Mail.Username, Password: cfg.Mail.Password, StartTLS: cfg.Mail.StartTLS, MaxPerSecond: cfg.Mail.MaxPerSecond, MaxPerHour: cfg.Mail.MaxPerHour})
		if err != nil {
			return err
		}
		smtpSender = decorateEmailSender(sender)
		notifier := newAuthNotifier(smtpSender, cfg.HTTP.PublicOrigin)
		a.authHTTP.Notifier = notifier
		a.authHTTP.InviteNotice = notifier
	} else if cfg.Mail.From != "" && !cfg.SafeMode {
		// Without a submission provider, the configured sender address opts into
		// direct STARTTLS delivery to recipient MX servers. Unavailable direct
		// delivery remains in the durable mail spool for retry.
		sender := mailer.MXSender{From: cfg.Mail.From}
		smtpSender = decorateEmailSender(sender)
		notifier := newAuthNotifier(smtpSender, cfg.HTTP.PublicOrigin)
		a.authHTTP.Notifier = notifier
		a.authHTTP.InviteNotice = notifier
	}
	a.smtpSender = smtpSender
	if modules["members"] && smtpSender != nil && strings.TrimSpace(cfg.HTTP.PublicOrigin) != "" {
		a.memberHTTP.Verification = members.EmailVerificationService{Tokens: a.memberStore}
		a.memberHTTP.VerificationTx = a.transactioner
		a.memberHTTP.VerificationNotifier = mailer.NotificationSender{Sender: smtpSender}
		a.memberHTTP.VerificationLimiter = a.newRateLimiter(3, time.Hour)
		a.memberHTTP.PublicOrigin = cfg.HTTP.PublicOrigin
		a.memberHTTP.VerificationEnabled = true
	}
	return nil
}

// initPublicCaches sets up the page and public HTML caches, the CGI static
// disk cache, the page content service and the SQLite external-write monitor.
func (a *serveApp) initPublicCaches() error {
	ctx, cfg, db, modules, stderr := a.ctx, a.cfg, a.db, a.modules, a.stderr
	requireTenantScope, budget := a.requireTenantScope, a.budget
	publicPageCache, err := newPageCache(modules["pages"], budget, a.redisClient, cfg.Redis.KeyPrefix)
	if err != nil {
		return fmt.Errorf("configure redis page cache: %w", err)
	}
	publicNewsCache, err := newNewsCache(modules["news"], budget, a.redisClient, cfg.Redis.KeyPrefix)
	if err != nil {
		return fmt.Errorf("configure redis news cache: %w", err)
	}
	var publicHTMLCache *server.PublicHTMLCache
	if modules["pages"] || modules["news"] {
		publicHTMLCache = server.NewPublicHTMLCache(server.PublicHTMLCacheOptions{
			MaxEntries: budget.PublicHTMLCacheEntries,
			MaxBytes:   budget.PublicHTMLCacheMaxBytes,
			MaxBody:    budget.PublicHTMLCacheMaxBodyBytes,
			TTL:        budget.CacheTTL,
		})
	}
	staticCacheDir := ""
	if a.opts.cgi && (modules["pages"] || modules["news"]) {
		if documentRoot := strings.TrimSpace(os.Getenv("DOCUMENT_ROOT")); documentRoot != "" {
			staticCacheDir = filepath.Join(documentRoot, "static_cache")
		}
	}
	a.publicPageCache, a.publicNewsCache, a.publicHTMLCache, a.staticCacheDir = publicPageCache, publicNewsCache, publicHTMLCache, staticCacheDir
	contentService := pageservice.ContentService{
		Transactions: a.transactioner,
		Pages:        pagesstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope},
		Audit:        auditstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope},
		Outbox:       outboxstore.Store{Dialect: cfg.Database.Dialect, RequireTenantScope: requireTenantScope},
		PublicCache:  publicPageCache,
		Realtime:     a.realtimeHub,
		Sanitize:     richtext.NewSanitizer().Sanitize,
	}
	if staticCacheDir != "" || publicHTMLCache != nil {
		contentService.PublicRoutePurge = func(locale, slug string) {
			path := "/public/pages/" + url.PathEscape(locale) + "/" + url.PathEscape(slug) + "/html"
			if publicHTMLCache != nil {
				publicHTMLCache.PurgePath(path)
			}
			if staticCacheDir != "" {
				if err := server.PurgeStaticDiskCachePath(staticCacheDir, path); err != nil {
					slog.Warn("published page cache invalidation failed", "error", err)
				}
			}
		}
	}
	a.contentService = contentService
	var sqliteCacheConn *sql.Conn
	if !a.opts.cgi && cfg.Database.SQLiteExternalWriteWatch && cfg.Database.Dialect == database.SQLite && (publicPageCache != nil || publicNewsCache != nil || publicHTMLCache != nil) {
		sqliteCacheConn, err = db.Conn(ctx)
		if err != nil {
			return fmt.Errorf("open sqlite cache monitor connection: %w", err)
		}
		a.onClose(func() error { _ = sqliteCacheConn.Close(); return nil })
		monitor := cache.NewSQLiteDataVersionMonitor(db)
		monitor.ReadVersion = func(readCtx context.Context) (int64, error) {
			var version int64
			err := sqliteCacheConn.QueryRowContext(readCtx, "PRAGMA data_version").Scan(&version)
			return version, err
		}
		monitor.OnChange = func() { clearPublicCaches(publicPageCache, publicNewsCache, publicHTMLCache) }
		monitor.OnError = func(monitorErr error) {
			if stderr != nil {
				_, _ = fmt.Fprintf(stderr, "hydro sqlite cache monitor: %v\n", monitorErr)
			}
		}
		a.spawn(func() {
			if monitorErr := monitor.Run(ctx); monitorErr != nil && ctx.Err() == nil && stderr != nil {
				_, _ = fmt.Fprintf(stderr, "hydro sqlite cache monitor stopped: %v\n", monitorErr)
			}
		})
	}
	return nil
}

// startScheduledBackup uploads a daily signed snapshot to S3-compatible
// storage when cloud backups are enabled.
func (a *serveApp) startScheduledBackup() error {
	ctx, cfg, db := a.ctx, a.cfg, a.db
	if cfg.Backup.Enabled && !cfg.SafeMode && !a.opts.cgi && !a.opts.fastCGI {
		dispatcher, backupErr := backupstore.NewS3Dispatcher(backupstore.S3Config{Endpoint: cfg.Media.S3Endpoint, Bucket: cfg.Media.S3Bucket, Region: cfg.Media.S3Region, AccessKey: cfg.Media.S3AccessKey, SecretKey: cfg.Media.S3SecretKey, Prefix: cfg.Backup.Prefix, Secure: cfg.Media.S3UseSSL, ForcePathStyle: cfg.Media.S3ForcePathStyle})
		if backupErr != nil {
			return fmt.Errorf("configure scheduled cloud backup: %w", backupErr)
		}
		a.spawn(func() {
			backupstore.RunDaily(ctx, cfg.Backup.HourUTC, func(backupCtx context.Context) error {
				root, err := os.MkdirTemp("", "hydrogo-backup-")
				if err != nil {
					return err
				}
				defer os.RemoveAll(root)
				snapshot := filepath.Join(root, "snapshot")
				if cfg.Database.Dialect == database.SQLite {
					_, err = backupstore.SnapshotSQLite(backupCtx, db, cfg.Media.Dir, snapshot)
				} else if cfg.Database.Dialect == database.Postgres {
					var tools backupstore.PostgresTools
					tools, err = backupstore.LocatePostgresTools()
					if err == nil {
						_, err = backupstore.SnapshotPostgres(backupCtx, tools, cfg.Database.DSN, cfg.Media.Dir, snapshot)
					}
				} else {
					err = errors.New("scheduled cloud backups support SQLite and PostgreSQL")
				}
				if err != nil {
					return err
				}
				name := fmt.Sprintf("hydrogo-%s.tar.gz", time.Now().UTC().Format("20060102T150405Z"))
				if err = dispatcher.UploadSnapshot(backupCtx, snapshot, name); err != nil {
					return err
				}
				_, err = dispatcher.Prune(backupCtx, time.Duration(cfg.Backup.RetentionDays)*24*time.Hour, time.Now().UTC())
				return err
			}, func(err error) { slog.Error("scheduled cloud backup failed", "error", err) })
		})
	}
	return nil
}

// newAuthNotifier sends password-reset and invitation e-mails that carry a
// link to the admin panel page redeeming the token. Without a public origin
// there is no address to link to, so the message falls back to the bare token.
func newAuthNotifier(sender mailer.Sender, publicOrigin string) mailer.NotificationSender {
	n := mailer.NotificationSender{Sender: sender}
	if strings.TrimSpace(publicOrigin) != "" {
		n.ResetLink = func(token string) string { return auth.PasswordResetLink(publicOrigin, token) }
		n.InviteLink = func(token string) string { return auth.InviteAcceptLink(publicOrigin, token) }
	}
	return n
}
