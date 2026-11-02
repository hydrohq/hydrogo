package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/cron"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/eventcatalog"
	"github.com/ardaozyildirim/hydrogo/internal/requestmeta"
)

var hookID = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,39}$`)

func (c Config) Validate() error {
	if !validTokenTTL(c.Security.PasswordResetTTL, []time.Duration{15 * time.Minute, time.Hour, 24 * time.Hour}) || !validTokenTTL(c.Security.InvitationTTL, []time.Duration{24 * time.Hour, 72 * time.Hour, 7 * 24 * time.Hour}) {
		return errors.New("security token lifetimes must be one of the supported choices")
	}
	if len(c.GeoIP.TrustedProxyCIDRs) > 256 || strings.ContainsAny(c.GeoIP.DatabasePath, "\x00\r\n") || len(c.GeoIP.DatabasePath) > 4096 {
		return errors.New("invalid GeoIP configuration")
	}
	for _, value := range c.GeoIP.TrustedProxyCIDRs {
		if _, _, err := net.ParseCIDR(strings.TrimSpace(value)); err != nil {
			return errors.New("invalid GeoIP trusted proxy CIDR")
		}
	}
	if c.Security.PreviewSigningKey != "" && len(c.Security.PreviewSigningKey) < 32 {
		return errors.New("preview signing key must contain at least 32 bytes")
	}
	if c.Backup.Enabled {
		hasS3 := c.Media.S3Endpoint != "" && c.Media.S3Bucket != "" && c.Media.S3AccessKey != "" && c.Media.S3SecretKey != ""
		hasLocal := strings.TrimSpace(c.Backup.LocalDir) != ""
		if !hasS3 && !hasLocal {
			return errors.New("scheduled backup requires configured S3-compatible storage credentials or a local backup directory")
		}
	}
	if c.Backup.Enabled && (c.Backup.RetentionDays < 1 || c.Backup.RetentionDays > 3650 || c.Backup.HourUTC < 0 || c.Backup.HourUTC > 23 || strings.ContainsAny(c.Backup.Prefix, "..\\")) {
		return errors.New("invalid scheduled backup retention, UTC hour, or prefix")
	}
	if (c.Notifications.Telegram.BotToken == "") != (c.Notifications.Telegram.ChatID == "") {
		return errors.New("Telegram notifications require both a bot token and a chat id")
	}
	if c.Commerce.Tax.Rate < 0 || c.Commerce.Tax.Rate >= 1 {
		return errors.New("commerce tax rate must be 0 (disabled) or a fraction below 1")
	}
	if c.Commerce.Shipping.FlatRateCents < 0 || c.Commerce.Shipping.FreeAboveCents < 0 {
		return errors.New("commerce shipping rates cannot be negative")
	}
	for _, currency := range c.Commerce.Currencies.Display {
		if c.Commerce.Currencies.Default == "" {
			return errors.New("commerce display currencies require a default currency")
		}
		if rate, ok := c.Commerce.Currencies.Rates[currency]; !ok || rate <= 0 {
			return fmt.Errorf("commerce display currency %q needs a positive rate", currency)
		}
	}
	if c.Backup.SigningSeed != "" {
		seed, err := hex.DecodeString(strings.TrimSpace(c.Backup.SigningSeed))
		if err != nil || len(seed) != 32 {
			return errors.New("backup signing seed must be 64 hexadecimal characters")
		}
	}
	if len(c.HTTP.TrustedProxies) > 64 {
		return errors.New("http.trusted_proxies allows at most 64 entries")
	}
	if _, err := requestmeta.NewIPExtractor(c.HTTP.BehindProxy, c.HTTP.TrustedProxies); err != nil {
		return err
	}
	if c.HTTP.AutoTLS {
		if c.HTTP.BehindProxy {
			return errors.New("automatic TLS cannot run behind a reverse proxy")
		}
		if strings.TrimSpace(c.HTTP.Domain) == "" || strings.ContainsAny(c.HTTP.Domain, "/:* ") {
			return errors.New("automatic TLS requires one valid domain")
		}
	}
	if err := c.Profile.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(c.I18N.DefaultLocale) == "" {
		return errors.New("i18n default locale cannot be empty")
	}
	if len(c.I18N.Locales) == 0 || len(c.I18N.Locales) > 64 {
		return errors.New("i18n locale list must contain 1-64 locales")
	}
	seenLocales := make(map[string]struct{}, len(c.I18N.Locales))
	defaultLocale := strings.ToLower(strings.TrimSpace(c.I18N.DefaultLocale))
	for _, locale := range c.I18N.Locales {
		locale = strings.TrimSpace(locale)
		key := strings.ToLower(locale)
		if !localePattern.MatchString(locale) {
			return fmt.Errorf("invalid i18n locale %q", locale)
		}
		if _, exists := seenLocales[key]; exists {
			return fmt.Errorf("i18n locale %q is configured more than once", locale)
		}
		seenLocales[key] = struct{}{}
	}
	if _, ok := seenLocales[defaultLocale]; !ok {
		return fmt.Errorf("i18n default locale %q is not enabled", c.I18N.DefaultLocale)
	}
	seenModules := make(map[string]struct{}, len(c.Modules))
	for _, module := range c.Modules {
		module = strings.TrimSpace(module)
		if module == "" {
			return errors.New("module name cannot be empty")
		}
		if _, exists := seenModules[module]; exists {
			return fmt.Errorf("module %q is configured more than once", module)
		}
		seenModules[module] = struct{}{}
	}
	if c.Tenancy.Enabled {
		if len(c.Tenancy.Hosts) == 0 || len(c.Tenancy.Hosts) > 4096 {
			return errors.New("enabled tenancy requires 1-4096 host mappings")
		}
		for host, tenantID := range c.Tenancy.Hosts {
			host = strings.ToLower(strings.TrimSpace(host))
			if host == "" || strings.ContainsAny(host, " /\\:") || !validConfigComponent(tenantID) {
				return fmt.Errorf("invalid tenancy host mapping %q", host)
			}
		}
		c.Tenancy.RequireHost = true
	}
	seenHooks := map[string]bool{}
	for _, hook := range c.Hooks {
		if !hookID.MatchString(hook.ID) || seenHooks[hook.ID] {
			return fmt.Errorf("hook id %q must be unique lowercase letters, digits, dashes or underscores", hook.ID)
		}
		seenHooks[hook.ID] = true
		parsed, err := url.Parse(strings.TrimSpace(hook.URL))
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Opaque != "" || parsed.Fragment != "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return fmt.Errorf("invalid URL for hook %q", hook.ID)
		}
		if strings.TrimSpace(hook.SecretEnv) == "" || len(hook.Secret) < 16 {
			return fmt.Errorf("hook %q needs secret_env naming an environment variable with a secret of at least 16 characters", hook.ID)
		}
		if hook.Timeout != "" {
			if timeout, err := time.ParseDuration(hook.Timeout); err != nil || timeout < 100*time.Millisecond || timeout > 10*time.Second {
				return fmt.Errorf("hook %q timeout must be between 100ms and 10s", hook.ID)
			}
		}
	}
	for _, hook := range c.Hooks {
		if len(hook.Schedules) > 20 {
			return fmt.Errorf("hook %q has more than 20 schedules", hook.ID)
		}
		names := map[string]bool{}
		for _, schedule := range hook.Schedules {
			if !hookID.MatchString(schedule.Name) || names[schedule.Name] {
				return fmt.Errorf("hook %q schedule name %q must be unique lowercase letters, digits, dashes or underscores", hook.ID, schedule.Name)
			}
			names[schedule.Name] = true
			if _, err := cron.Parse(schedule.Cron); err != nil {
				return fmt.Errorf("hook %q schedule %q: %w", hook.ID, schedule.Name, err)
			}
		}
	}
	for _, webhook := range c.Webhooks {
		if strings.TrimSpace(webhook.ID) == "" || strings.TrimSpace(webhook.URL) == "" || strings.TrimSpace(webhook.Secret) == "" {
			return errors.New("webhook id, url and environment secret are required")
		}
		parsed, err := url.Parse(strings.TrimSpace(webhook.URL))
		if err != nil || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil || parsed.Opaque != "" || parsed.Fragment != "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return fmt.Errorf("invalid webhook URL %q", webhook.URL)
		}
		if len(webhook.Topics) > 64 {
			return errors.New("webhook topics exceed limit")
		}
		seenTopics := make(map[string]struct{}, len(webhook.Topics))
		for _, topic := range webhook.Topics {
			topic = strings.TrimSpace(topic)
			if topic == "" {
				return errors.New("webhook topic cannot be empty")
			}
			if _, exists := seenTopics[topic]; exists {
				return fmt.Errorf("webhook topic %q is configured more than once", topic)
			}
			seenTopics[topic] = struct{}{}
			known, pii := eventcatalog.TopicPolicy(topic)
			if !known {
				return fmt.Errorf("webhook topic %q is not published in the event catalog", topic)
			}
			if pii && !webhook.AllowPII {
				return fmt.Errorf("webhook topic %q requires explicit allow_pii", topic)
			}
		}
	}
	if _, _, err := net.SplitHostPort(c.HTTP.Address); err != nil {
		return fmt.Errorf("invalid HTTP address %q: %w", c.HTTP.Address, err)
	}
	var publicOrigin *url.URL
	if strings.TrimSpace(c.HTTP.PublicOrigin) != "" {
		parsed, err := url.Parse(c.HTTP.PublicOrigin)
		if err != nil || !isHTTPOrigin(parsed) {
			return fmt.Errorf("invalid public origin %q", c.HTTP.PublicOrigin)
		}
		publicOrigin = parsed
	}
	if c.Security.EnableHSTS && (publicOrigin == nil || publicOrigin.Scheme != "https") {
		return errors.New("HSTS requires an HTTPS public origin")
	}
	if c.Security.EnableHSTS && !c.Security.SecureCookies {
		return errors.New("HSTS requires secure cookies")
	}
	if preset := strings.ToLower(strings.TrimSpace(c.Security.HeadersPreset)); preset != "" && preset != "strict" && preset != "standard" && preset != "relaxed" {
		return errors.New("security headers preset must be strict, standard or relaxed")
	}
	if len(c.Security.AnomalyBypassPaths) > 32 {
		return errors.New("security anomaly bypass path list exceeds limit")
	}
	if c.AdminSupport.Enabled {
		if strings.TrimSpace(c.AdminSupport.AgencyName) == "" || len(c.AdminSupport.AgencyName) > 120 || len(c.AdminSupport.SLABadge) > 80 {
			return errors.New("admin support requires an agency name and bounded label")
		}
		if c.AdminSupport.Email != "" {
			address, err := mail.ParseAddress(strings.TrimSpace(c.AdminSupport.Email))
			if err != nil || address.Address != strings.TrimSpace(c.AdminSupport.Email) {
				return errors.New("invalid admin support email")
			}
		}
		if strings.ContainsAny(c.AdminSupport.Phone, "<>\"'") || len(c.AdminSupport.Phone) > 40 {
			return errors.New("invalid admin support phone")
		}
		if c.AdminSupport.Email == "" && c.AdminSupport.Phone == "" && c.AdminSupport.WhatsAppURL == "" {
			return errors.New("admin support requires at least one contact method")
		}
	}
	for _, raw := range []string{c.AdminSupport.WhatsAppURL, c.AdminSupport.GuideURL} {
		if raw == "" {
			continue
		}
		parsed, err := url.Parse(strings.TrimSpace(raw))
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
			return errors.New("admin support links must be valid HTTPS URLs")
		}
	}
	for _, path := range c.Security.AnomalyBypassPaths {
		if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.ContainsAny(path, "?#\\") {
			return fmt.Errorf("invalid security anomaly bypass path %q", path)
		}
	}
	if len(c.Security.MFARequiredRoles) > 3 {
		return errors.New("MFA required role list exceeds limit")
	}
	for _, role := range c.Security.MFARequiredRoles {
		if role != "admin" && role != "editor" && role != "viewer" {
			return fmt.Errorf("invalid MFA required role %q", role)
		}
	}
	if c.Security.SecureCookies && (publicOrigin == nil || publicOrigin.Scheme != "https") {
		return errors.New("secure cookies require an HTTPS public origin")
	}
	if publicOrigin != nil && publicOrigin.Scheme == "https" && !c.Security.SecureCookies {
		return errors.New("HTTPS public origin requires secure cookies")
	}
	seenOrigins := make(map[string]struct{}, len(c.Security.PublicAPIOrigins))
	if len(c.Security.PublicAPIOrigins) > 32 {
		return errors.New("public API origin allowlist exceeds limit")
	}
	for _, origin := range c.Security.PublicAPIOrigins {
		parsed, err := url.Parse(strings.TrimSpace(origin))
		if err != nil || !isHTTPOrigin(parsed) {
			return fmt.Errorf("invalid public API origin %q", origin)
		}
		key := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(origin)), "/")
		if _, exists := seenOrigins[key]; exists {
			return fmt.Errorf("public API origin %q is configured more than once", origin)
		}
		seenOrigins[key] = struct{}{}
	}
	switch c.Database.Dialect {
	case database.SQLite, database.Postgres, database.MySQL:
	default:
		return fmt.Errorf("unsupported database dialect %q", c.Database.Dialect)
	}
	if err := database.ValidateDialectAvailable(c.Database.Dialect); err != nil {
		return err
	}
	if strings.TrimSpace(c.Database.DSN) == "" {
		return errors.New("database DSN is required")
	}
	if strings.TrimSpace(c.Media.Dir) == "" {
		return errors.New("media directory is required")
	}
	if c.Media.MinFreeBytes > 1<<50 {
		return errors.New("media minimum free space exceeds limit")
	}
	if c.Media.WatermarkText != "" {
		position := c.Media.WatermarkPosition
		if position != "" && position != "bottom-right" && position != "bottom-left" && position != "top-right" && position != "center" {
			return errors.New("invalid media watermark position")
		}
		if c.Media.WatermarkOpacity != 0 && (c.Media.WatermarkOpacity < .1 || c.Media.WatermarkOpacity > 1) {
			return errors.New("media watermark opacity must be between 0.1 and 1")
		}
		if c.Media.WatermarkMargin < 0 || c.Media.WatermarkMargin > 1000 {
			return errors.New("invalid media watermark margin")
		}
	}
	for _, host := range c.Media.AllowedRefererHosts {
		host = strings.TrimSpace(host)
		if host == "" || strings.ContainsAny(host, "/\\@:") {
			return errors.New("invalid media referer host")
		}
		parsed, err := url.Parse("https://" + host)
		if err != nil || parsed.Hostname() == "" || parsed.Host != host {
			return errors.New("invalid media referer host")
		}
	}
	if c.Media.StorageDriver == "" {
		c.Media.StorageDriver = "local"
	}
	if c.Media.StorageDriver != "local" && c.Media.StorageDriver != "s3" {
		return fmt.Errorf("unsupported media storage driver %q", c.Media.StorageDriver)
	}
	if err := validateMediaStorageBuild(c.Media.StorageDriver); err != nil {
		return err
	}
	if err := validateRedisBuild(c.Redis.URL); err != nil {
		return err
	}
	if c.Media.StorageDriver == "s3" && (strings.TrimSpace(c.Media.S3Endpoint) == "" || strings.TrimSpace(c.Media.S3Bucket) == "") {
		return errors.New("s3 media storage requires endpoint and bucket")
	}
	if c.Media.CDNBaseURL != "" {
		parsed, err := url.Parse(c.Media.CDNBaseURL)
		if err != nil || !isHTTPBaseURL(parsed) {
			return fmt.Errorf("invalid media CDN base URL %q", c.Media.CDNBaseURL)
		}
	}
	if c.License.Required {
		if strings.TrimSpace(c.License.File) == "" || strings.TrimSpace(c.License.ProjectID) == "" || strings.TrimSpace(c.License.Edition) == "" {
			return errors.New("required license needs file, project_id and edition")
		}
		if len(c.License.Features) > 64 {
			return errors.New("license feature list exceeds limit")
		}
	}
	if c.Mail.Host == "" {
		if c.Mail.Username != "" || c.Mail.Password != "" {
			return errors.New("mail username and password require an SMTP host")
		}
	}
	if c.Mail.From != "" {
		from := strings.TrimSpace(c.Mail.From)
		parsed, err := mail.ParseAddress(from)
		if err != nil || parsed.Address != from || strings.ContainsAny(from, "\r\n") {
			return errors.New("mail from is invalid")
		}
	}
	if c.Mail.Host != "" {
		if strings.TrimSpace(c.Mail.From) == "" {
			return errors.New("mail from is required when SMTP is configured")
		}
		if c.Mail.Port <= 0 || c.Mail.Port > 65535 {
			return errors.New("mail port is invalid")
		}
		if c.Mail.Username == "" && c.Mail.Password != "" {
			return errors.New("mail password requires username")
		}
		if c.Mail.Username != "" && !c.Mail.StartTLS {
			return errors.New("mail SMTP AUTH requires STARTTLS")
		}
	}
	if c.Mail.MaxPerSecond < 0 || c.Mail.MaxPerSecond > 3600 || c.Mail.MaxPerHour < 0 || c.Mail.MaxPerHour > 100_000 {
		return errors.New("SMTP rate limits are invalid")
	}
	if (c.Forms.TurnstileSiteKey == "") != (c.Forms.TurnstileSecretKey == "") {
		return errors.New("Turnstile site key and secret key must be configured together")
	}
	if len(c.Forms.TurnstileSiteKey) > 256 || len(c.Forms.TurnstileSecretKey) > 512 || strings.ContainsAny(c.Forms.TurnstileSiteKey+c.Forms.TurnstileSecretKey, "\r\n") {
		return errors.New("Turnstile configuration is invalid")
	}
	if len(c.SEO.OGCacheDir) > 4096 || len(c.SEO.FontsDir) > 4096 || strings.ContainsAny(c.SEO.OGCacheDir+c.SEO.FontsDir, "\r\n\x00") {
		return errors.New("SEO asset directory is invalid")
	}
	if len(c.Analytics.HashKey) > 4096 || strings.ContainsAny(c.Analytics.HashKey, "\r\n\x00") {
		return errors.New("analytics hash key is invalid")
	}
	if len(c.Observability.SentryDSN) > 4096 || strings.ContainsAny(c.Observability.SentryDSN, "\r\n\x00") {
		return errors.New("Sentry configuration is invalid")
	}
	if c.Analytics.NoCookieEnabled {
		analyticsModule := false
		for _, module := range c.Modules {
			if module == "analytics" {
				analyticsModule = true
				break
			}
		}
		if !analyticsModule {
			return errors.New("no-cookie analytics requires the analytics module")
		}
		if len(c.Analytics.HashKey) < 32 {
			return errors.New("no-cookie analytics requires an analytics hash key of at least 32 bytes")
		}
	}
	return nil
}
