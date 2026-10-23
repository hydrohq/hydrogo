package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"gopkg.in/yaml.v3"
)

type LookupEnv func(string) (string, bool)

func Load(path string, lookup LookupEnv, overrides Overrides) (Config, error) {
	if lookup == nil {
		lookup = func(string) (string, bool) { return "", false }
	}
	result := Config{Profile: capacity.Compact, Modules: []string{"core", "auth", "members", "pages", "media", "site", "news", "collections"}, HTTP: HTTP{Address: "127.0.0.1:8080"}, Database: Database{Dialect: database.SQLite}, Media: Media{Dir: "data/media", MinFreeBytes: 50 << 20}, Backup: Backup{Prefix: "hydrogo", RetentionDays: 30, HourUTC: 3}, ContentTypesDir: "content-types", SEO: SEO{FontsDir: "public/fonts"}, Mail: Mail{Port: 587, StartTLS: true}, I18N: I18N{Enabled: false, DefaultLocale: "tr", Locales: []string{"tr"}}, Security: Security{EnableAnomalyFilter: true, PasswordResetTTL: "1h", InvitationTTL: "24h"}}
	result.License.File = "hydro.lic"
	if path != "" {
		file, err := os.Open(path) // #nosec G304 -- the operator's config file path
		if err != nil {
			return Config{}, err
		}
		defer file.Close()
		decoder := yaml.NewDecoder(file)
		decoder.KnownFields(true)
		if err := decoder.Decode(&result); err != nil {
			return Config{}, fmt.Errorf("decode config: %w", err)
		}
	}
	if result.Security.PasswordResetTTL == "" {
		result.Security.PasswordResetTTL = "1h"
	}
	if result.Security.InvitationTTL == "" {
		result.Security.InvitationTTL = "24h"
	}

	if value, ok := lookup("HYDRO_HTTP_ADDRESS"); ok {
		result.HTTP.Address = value
	}
	if value, ok := lookup("HYDRO_SAFE_MODE"); ok && strings.TrimSpace(value) != "" {
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_SAFE_MODE: %w", err)
		}
		result.SafeMode = parsed
	}
	if value, ok := lookup("HYDRO_CAPACITY_PROFILE"); ok && value != "" {
		result.Profile = capacity.Profile(value)
	}
	if value, ok := lookup("HYDRO_PUBLIC_ORIGIN"); ok && value != "" {
		result.HTTP.PublicOrigin = value
	}
	if value, ok := lookup("HYDRO_DATABASE_DIALECT"); ok {
		result.Database.Dialect = database.Dialect(value)
	}
	if value, ok := lookup("HYDRO_MEDIA_DIR"); ok && value != "" {
		result.Media.Dir = value
	}
	if value, ok := lookup("HYDRO_MEDIA_MIN_FREE_BYTES"); ok && value != "" {
		minFree, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_MEDIA_MIN_FREE_BYTES: %w", err)
		}
		result.Media.MinFreeBytes = minFree
	}
	if value, ok := lookup("HYDRO_MEDIA_STORAGE_DRIVER"); ok && value != "" {
		result.Media.StorageDriver = value
	}
	if value, ok := lookup("HYDRO_MEDIA_S3_ENDPOINT"); ok && value != "" {
		result.Media.S3Endpoint = value
	}
	if value, ok := lookup("HYDRO_MEDIA_S3_BUCKET"); ok && value != "" {
		result.Media.S3Bucket = value
	}
	if value, ok := lookup("HYDRO_MEDIA_S3_REGION"); ok && value != "" {
		result.Media.S3Region = value
	}
	if value, ok := lookup("HYDRO_MEDIA_S3_ACCESS_KEY"); ok && value != "" {
		result.Media.S3AccessKey = value
	}
	if value, ok := lookup("HYDRO_MEDIA_S3_SECRET_KEY"); ok && value != "" {
		result.Media.S3SecretKey = value
	}
	if value, ok := lookup("HYDRO_MEDIA_S3_USE_SSL"); ok && value != "" {
		useSSL, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_MEDIA_S3_USE_SSL: %w", err)
		}
		result.Media.S3UseSSL = useSSL
	}
	if value, ok := lookup("HYDRO_MEDIA_S3_FORCE_PATH_STYLE"); ok && value != "" {
		forcePathStyle, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_MEDIA_S3_FORCE_PATH_STYLE: %w", err)
		}
		result.Media.S3ForcePathStyle = forcePathStyle
	}
	if value, ok := lookup("HYDRO_MEDIA_CDN_BASE_URL"); ok && value != "" {
		result.Media.CDNBaseURL = value
	}
	if value, ok := lookup("HYDRO_MEDIA_HOTLINK_PROTECTION"); ok && value != "" {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_MEDIA_HOTLINK_PROTECTION: %w", err)
		}
		result.Media.HotlinkProtection = enabled
	}
	if value, ok := lookup("HYDRO_MEDIA_ALLOWED_REFERER_HOSTS"); ok {
		for _, host := range strings.Split(value, ",") {
			if host = strings.TrimSpace(host); host != "" {
				result.Media.AllowedRefererHosts = append(result.Media.AllowedRefererHosts, host)
			}
		}
	}
	if value, ok := lookup("HYDRO_MEDIA_WATERMARK_TEXT"); ok {
		result.Media.WatermarkText = value
	}
	if value, ok := lookup("HYDRO_MEDIA_WATERMARK_POSITION"); ok && value != "" {
		result.Media.WatermarkPosition = value
	}
	if value, ok := lookup("HYDRO_MEDIA_WATERMARK_OPACITY"); ok && value != "" {
		opacity, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_MEDIA_WATERMARK_OPACITY: %w", err)
		}
		result.Media.WatermarkOpacity = opacity
	}
	if value, ok := lookup("HYDRO_MEDIA_WATERMARK_MARGIN"); ok && value != "" {
		margin, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_MEDIA_WATERMARK_MARGIN: %w", err)
		}
		result.Media.WatermarkMargin = margin
	}
	if value, ok := lookup("HYDRO_CONTENT_TYPES_DIR"); ok && value != "" {
		result.ContentTypesDir = value
	}
	if value, ok := lookup("HYDRO_BACKUP_SIGNING_SEED"); ok && value != "" {
		result.Backup.SigningSeed = value
	}
	if value, ok := lookup("HYDRO_ADMIN_EXTENSION_MANIFEST_DIR"); ok && value != "" {
		result.AdminExtensions.ManifestDir = value
	}
	if value, ok := lookup("HYDRO_ADMIN_EXTENSION_TRUSTED_KEYS_FILE"); ok && value != "" {
		result.AdminExtensions.TrustedKeysFile = value
	}
	if value, ok := lookup("HYDRO_I18N_ENABLED"); ok && value != "" {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_I18N_ENABLED: %w", err)
		}
		result.I18N.Enabled = enabled
	}
	if value, ok := lookup("HYDRO_DEFAULT_LOCALE"); ok && value != "" {
		result.I18N.DefaultLocale = value
	}
	if value, ok := lookup("HYDRO_LOCALES"); ok && value != "" {
		var locales []string
		for _, item := range strings.Split(value, ",") {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				locales = append(locales, trimmed)
			}
		}
		if len(locales) > 0 {
			result.I18N.Locales = locales
		}
	}
	for index := range result.Hooks {
		if name := strings.TrimSpace(result.Hooks[index].SecretEnv); name != "" {
			result.Hooks[index].Secret, _ = lookup(name)
		}
	}
	if value, ok := lookup("HYDRO_WEBHOOK_URL"); ok && value != "" {
		if len(result.Webhooks) == 0 {
			result.Webhooks = []Webhook{{ID: "default", URL: value}}
		} else {
			result.Webhooks[0].URL = value
		}
	}
	if value, ok := lookup("HYDRO_WEBHOOK_SECRET"); ok && value != "" {
		if len(result.Webhooks) == 0 {
			result.Webhooks = []Webhook{{ID: "default"}}
		}
		result.Webhooks[0].Secret = value
	}
	if value, ok := lookup("HYDRO_WEBHOOK_TOPICS"); ok && value != "" {
		if len(result.Webhooks) == 0 {
			result.Webhooks = []Webhook{{ID: "default"}}
		}
		result.Webhooks[0].Topics = strings.Split(value, ",")
	}
	if value, ok := lookup("HYDRO_MFA_MASTER_KEY"); ok && value != "" {
		result.Security.MFAMasterKey = value
	}
	if value, ok := lookup("HYDRO_PREVIEW_SIGNING_KEY"); ok && value != "" {
		result.Security.PreviewSigningKey = value
	}
	if value, ok := lookup("HYDRO_PASSWORD_RESET_TTL"); ok && value != "" {
		result.Security.PasswordResetTTL = value
	}
	if value, ok := lookup("HYDRO_INVITATION_TTL"); ok && value != "" {
		result.Security.InvitationTTL = value
	}
	if value, ok := lookup("HYDRO_SECURE_COOKIES"); ok && value != "" {
		secureCookies, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_SECURE_COOKIES: %w", err)
		}
		result.Security.SecureCookies = secureCookies
	}
	if value, ok := lookup("HYDRO_ENABLE_HSTS"); ok && value != "" {
		enableHSTS, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_ENABLE_HSTS: %w", err)
		}
		result.Security.EnableHSTS = enableHSTS
	}
	if value, ok := lookup("HYDRO_AUDIT_LOG_REQUEST_BODIES"); ok && value != "" {
		logBodies, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_AUDIT_LOG_REQUEST_BODIES: %w", err)
		}
		result.Audit.LogRequestBodies = logBodies
	}
	if value, ok := lookup("HYDRO_PUBLIC_API_ORIGINS"); ok && value != "" {
		result.Security.PublicAPIOrigins = strings.Split(value, ",")
	}
	if value, ok := lookup("HYDRO_REDIS_URL"); ok && value != "" {
		result.Redis.URL = value
	}
	if value, ok := lookup("HYDRO_REDIS_KEY_PREFIX"); ok && value != "" {
		result.Redis.KeyPrefix = value
	}
	if value, ok := lookup("HYDRO_LICENSE_REQUIRED"); ok && value != "" {
		required, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_LICENSE_REQUIRED: %w", err)
		}
		result.License.Required = required
	}
	if value, ok := lookup("HYDRO_LICENSE_FILE"); ok && value != "" {
		result.License.File = value
	}
	if value, ok := lookup("HYDRO_LICENSE_PROJECT_ID"); ok && value != "" {
		result.License.ProjectID = value
	}
	if value, ok := lookup("HYDRO_LICENSE_EDITION"); ok && value != "" {
		result.License.Edition = value
	}
	if value, ok := lookup("HYDRO_LICENSE_FEATURES"); ok && value != "" {
		result.License.Features = strings.Split(value, ",")
	}
	if value, ok := lookup("HYDRO_SMTP_HOST"); ok && value != "" {
		result.Mail.Host = value
	}
	if value, ok := lookup("HYDRO_SMTP_FROM"); ok && value != "" {
		result.Mail.From = value
	}
	if value, ok := lookup("HYDRO_SMTP_DKIM_SELECTOR"); ok && value != "" {
		result.Mail.DKIMSelector = value
	}
	if value, ok := lookup("HYDRO_SMTP_USERNAME"); ok && value != "" {
		result.Mail.Username = value
	}
	if value, ok := lookup("HYDRO_SMTP_PASSWORD"); ok && value != "" {
		result.Mail.Password = value
	}
	if value, ok := lookup("HYDRO_SMTP_PORT"); ok && value != "" {
		port, err := strconv.Atoi(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_SMTP_PORT: %w", err)
		}
		result.Mail.Port = port
	}
	if value, ok := lookup("HYDRO_SMTP_STARTTLS"); ok && value != "" {
		startTLS, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_SMTP_STARTTLS: %w", err)
		}
		result.Mail.StartTLS = startTLS
	}
	for _, setting := range []struct {
		name   string
		target *int
	}{{"HYDRO_SMTP_MAX_PER_SECOND", &result.Mail.MaxPerSecond}, {"HYDRO_SMTP_MAX_PER_HOUR", &result.Mail.MaxPerHour}} {
		if value, ok := lookup(setting.name); ok && value != "" {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return Config{}, fmt.Errorf("invalid %s: %w", setting.name, err)
			}
			*setting.target = parsed
		}
	}
	if value, ok := lookup("HYDRO_FORMS_TURNSTILE_SITE_KEY"); ok && value != "" {
		result.Forms.TurnstileSiteKey = value
	}
	if value, ok := lookup("HYDRO_FORMS_TURNSTILE_SECRET_KEY"); ok && value != "" {
		result.Forms.TurnstileSecretKey = value
	}
	if value, ok := lookup("HYDRO_SEO_OG_CACHE_DIR"); ok && value != "" {
		result.SEO.OGCacheDir = value
	}
	if value, ok := lookup("HYDRO_SEO_FONTS_DIR"); ok && value != "" {
		result.SEO.FontsDir = value
	}
	if value, ok := lookup("HYDRO_ANALYTICS_NO_COOKIE_ENABLED"); ok && value != "" {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_ANALYTICS_NO_COOKIE_ENABLED: %w", err)
		}
		result.Analytics.NoCookieEnabled = enabled
	}
	if value, ok := lookup("HYDRO_ANALYTICS_HASH_KEY"); ok && value != "" {
		result.Analytics.HashKey = value
	}
	if value, ok := lookup("HYDRO_SENTRY_DSN"); ok && value != "" {
		result.Observability.SentryDSN = value
	}
	dsnSources := 0
	if result.Database.DSN != "" {
		dsnSources++
	}
	if value, ok := lookup("HYDRO_DATABASE_DSN"); ok && value != "" {
		dsnSources++
		result.Database.DSN = value
	}
	if value, ok := lookup("HYDRO_SQLITE_EXTERNAL_WRITE_WATCH"); ok && value != "" {
		watch, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("invalid HYDRO_SQLITE_EXTERNAL_WRITE_WATCH: %w", err)
		}
		result.Database.SQLiteExternalWriteWatch = watch
	}
	if overrides.DatabaseDSN != nil && *overrides.DatabaseDSN != "" {
		dsnSources++
		result.Database.DSN = *overrides.DatabaseDSN
	}
	if overrides.SafeMode != nil {
		result.SafeMode = *overrides.SafeMode
	}
	if dsnSources > 1 {
		return Config{}, errors.New("database DSN is defined by multiple secret sources")
	}
	for index, module := range result.Modules {
		result.Modules[index] = strings.ToLower(strings.TrimSpace(module))
	}

	if overrides.HTTPAddress != nil {
		result.HTTP.Address = *overrides.HTTPAddress
	}
	if overrides.CapacityProfile != nil {
		result.Profile = capacity.Profile(*overrides.CapacityProfile)
	}
	if overrides.DatabaseDialect != nil {
		result.Database.Dialect = database.Dialect(*overrides.DatabaseDialect)
	}
	if result.Tenancy.Enabled {
		result.Tenancy.RequireHost = true
	}
	if overrides.SkipDeliverySecrets {
		const unused = "unused-by-this-command"
		for index := range result.Hooks {
			if result.Hooks[index].Secret == "" {
				result.Hooks[index].Secret = unused
			}
		}
		for index := range result.Webhooks {
			if result.Webhooks[index].Secret == "" {
				result.Webhooks[index].Secret = unused
			}
		}
	}
	if err := result.Validate(); err != nil {
		return Config{}, err
	}
	return result, nil
}
