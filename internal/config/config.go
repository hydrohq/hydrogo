package config

import (
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/eventcatalog"
)

var localePattern = regexp.MustCompile(`^[A-Za-z]{2,3}(?:-[A-Za-z]{4})?(?:-(?:[A-Za-z]{2}|[0-9]{3}))?$`)

type Config struct {
	Profile         capacity.Profile `yaml:"profile"`
	Modules         []string         `yaml:"modules"`
	HTTP            HTTP             `yaml:"http"`
	Database        Database         `yaml:"database"`
	Media           Media            `yaml:"media"`
	ThemeDir        string           `yaml:"theme_dir"`
	Theme           string           `yaml:"theme"`
	AdminExtensions AdminExtensions  `yaml:"admin_extensions"`
	AdminSupport    AdminSupport     `yaml:"admin_support"`
	Tenancy         Tenancy          `yaml:"tenancy"`
	I18N            I18N             `yaml:"i18n"`
	Security        Security         `yaml:"security"`
	Mail            Mail             `yaml:"mail"`
	Forms           Forms            `yaml:"forms"`
	SEO             SEO              `yaml:"seo"`
	Analytics       Analytics        `yaml:"analytics"`
	Observability   Observability    `yaml:"observability"`
	Redis           Redis            `yaml:"redis"`
	License         License          `yaml:"license"`
	Webhooks        []Webhook        `yaml:"webhooks"`
}

// AdminExtensions configures signed, externally hosted Admin Studio surfaces.
// Empty paths keep the feature disabled and impose no runtime overhead.
type AdminExtensions struct {
	ManifestDir     string `yaml:"manifest_dir"`
	TrustedKeysFile string `yaml:"trusted_keys_file"`
}

type AdminSupport struct {
	Enabled     bool   `yaml:"enabled"`
	AgencyName  string `yaml:"agency_name"`
	Email       string `yaml:"email"`
	Phone       string `yaml:"phone"`
	WhatsAppURL string `yaml:"whatsapp_url"`
	SLABadge    string `yaml:"sla_badge"`
	GuideURL    string `yaml:"guide_url"`
}

// Tenancy maps trusted request hosts to tenant IDs. IDs come only from
// operator configuration; request headers and query parameters are ignored.
type Tenancy struct {
	Enabled     bool              `yaml:"enabled"`
	RequireHost bool              `yaml:"require_host"`
	Hosts       map[string]string `yaml:"hosts"`
}

type Webhook struct {
	ID       string   `yaml:"id"`
	URL      string   `yaml:"url"`
	Secret   string   `yaml:"-"`
	Topics   []string `yaml:"topics"`
	AllowPII bool     `yaml:"allow_pii"`
}

type HTTP struct {
	Address      string `yaml:"address"`
	PublicOrigin string `yaml:"public_origin"`
}

type Database struct {
	Dialect                  database.Dialect `yaml:"dialect"`
	DSN                      string           `yaml:"dsn"`
	SQLiteExternalWriteWatch bool             `yaml:"sqlite_external_write_watch"`
}

type Media struct {
	Dir              string `yaml:"dir"`
	MinFreeBytes     uint64 `yaml:"min_free_bytes"`
	StorageDriver    string `yaml:"storage_driver"`
	S3Endpoint       string `yaml:"s3_endpoint"`
	S3Bucket         string `yaml:"s3_bucket"`
	S3Region         string `yaml:"s3_region"`
	S3AccessKey      string `yaml:"s3_access_key"`
	S3SecretKey      string `yaml:"s3_secret_key"`
	S3UseSSL         bool   `yaml:"s3_use_ssl"`
	S3ForcePathStyle bool   `yaml:"s3_force_path_style"`
	CDNBaseURL       string `yaml:"cdn_base_url"`
}

type Security struct {
	MFAMasterKey        string   `yaml:"mfa_master_key"`
	SecureCookies       bool     `yaml:"secure_cookies"`
	EnableHSTS          bool     `yaml:"enable_hsts"`
	HeadersPreset       string   `yaml:"headers_preset"`
	PublicAPIOrigins    []string `yaml:"public_api_origins"`
	EnableAnomalyFilter bool     `yaml:"enable_anomaly_filter"`
	AnomalyBypassPaths  []string `yaml:"anomaly_bypass_paths"`
	MFARequiredRoles    []string `yaml:"mfa_required_roles"`
}

type I18N struct {
	Enabled       bool     `yaml:"enabled"`
	DefaultLocale string   `yaml:"default_locale"`
	Locales       []string `yaml:"locales"`
}

type Mail struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	From     string `yaml:"from"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	StartTLS bool   `yaml:"starttls"`
}

type Forms struct {
	TurnstileSiteKey   string `yaml:"turnstile_site_key"`
	TurnstileSecretKey string `yaml:"turnstile_secret_key"`
}

type SEO struct {
	OGCacheDir string `yaml:"og_cache_dir"`
	FontsDir   string `yaml:"fonts_dir"`
}

// Analytics stays disabled unless an operator explicitly enables the
// no-cookie aggregate feature and provides a deployment-shared hash key.
type Analytics struct {
	NoCookieEnabled bool   `yaml:"no_cookie_enabled"`
	HashKey         string `yaml:"hash_key"`
}

// Observability integrations are inactive by default and accept only operator
// configured endpoints. DSNs are never returned by config summaries.
type Observability struct {
	SentryDSN string `yaml:"sentry_dsn"`
}

type Redis struct {
	URL       string `yaml:"url"`
	KeyPrefix string `yaml:"key_prefix"`
}

type License struct {
	Required  bool     `yaml:"required"`
	File      string   `yaml:"file"`
	ProjectID string   `yaml:"project_id"`
	Edition   string   `yaml:"edition"`
	Features  []string `yaml:"features"`
}

type Overrides struct {
	CapacityProfile *string
	HTTPAddress     *string
	DatabaseDialect *string
	DatabaseDSN     *string
}

type LookupEnv func(string) (string, bool)

func Load(path string, lookup LookupEnv, overrides Overrides) (Config, error) {
	if lookup == nil {
		lookup = func(string) (string, bool) { return "", false }
	}
	result := Config{Profile: capacity.Compact, Modules: []string{"core", "auth", "members", "pages", "media", "site", "news"}, HTTP: HTTP{Address: "127.0.0.1:8080"}, Database: Database{Dialect: database.SQLite}, Media: Media{Dir: "data/media", MinFreeBytes: 50 << 20}, ThemeDir: "themes", SEO: SEO{FontsDir: "public/fonts"}, Mail: Mail{Port: 587, StartTLS: true}, I18N: I18N{Enabled: false, DefaultLocale: "tr", Locales: []string{"tr"}}, Security: Security{EnableAnomalyFilter: true}}
	result.License.File = "hydro.lic"
	if path != "" {
		file, err := os.Open(path)
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

	if value, ok := lookup("HYDRO_HTTP_ADDRESS"); ok {
		result.HTTP.Address = value
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
	if value, ok := lookup("HYDRO_THEME_DIR"); ok && value != "" {
		result.ThemeDir = value
	}
	if value, ok := lookup("HYDRO_THEME"); ok && value != "" {
		result.Theme = value
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
	if err := result.Validate(); err != nil {
		return Config{}, err
	}
	return result, nil
}

func (c Config) Validate() error {
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
		if module == "commerce" {
			return errors.New("commerce is deferred and cannot be enabled in this release")
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

func isHTTPOrigin(parsed *url.URL) bool {
	if parsed == nil || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return (parsed.Path == "" || parsed.Path == "/") && parsed.RawQuery == "" && parsed.Fragment == ""
}

func isHTTPBaseURL(parsed *url.URL) bool {
	if parsed == nil || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return len(parsed.Path) <= 512 && parsed.RawQuery == "" && parsed.Fragment == ""
}

func isHTTPSCallbackURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && parsed != nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.Hostname() != "" && parsed.User == nil && parsed.Opaque == "" && parsed.Fragment == ""
}

func validConfigComponent(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}
