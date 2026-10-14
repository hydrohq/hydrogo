package config

import (
	"regexp"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var localePattern = regexp.MustCompile(`^[A-Za-z]{2,3}(?:-[A-Za-z]{4})?(?:-(?:[A-Za-z]{2}|[0-9]{3}))?$`)

type Config struct {
	SafeMode        bool             `yaml:"safe_mode"`
	Profile         capacity.Profile `yaml:"profile"`
	Modules         []string         `yaml:"modules"`
	HTTP            HTTP             `yaml:"http"`
	Database        Database         `yaml:"database"`
	Media           Media            `yaml:"media"`
	ThemeDir        string           `yaml:"theme_dir"`
	ContentTypesDir string           `yaml:"content_types_dir"`
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
	GeoIP           GeoIP            `yaml:"geoip"`
	Observability   Observability    `yaml:"observability"`
	Redis           Redis            `yaml:"redis"`
	License         License          `yaml:"license"`
	Webhooks        []Webhook        `yaml:"webhooks"`
	Backup          Backup           `yaml:"backup"`
}

type Backup struct {
	Enabled       bool   `yaml:"enabled"`
	Prefix        string `yaml:"prefix"`
	RetentionDays int    `yaml:"retention_days"`
	HourUTC       int    `yaml:"hour_utc"`
	SigningSeed   string `yaml:"-" json:"-"`
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
	AutoTLS      bool   `yaml:"auto_tls"`
	Domain       string `yaml:"domain"`
	BehindProxy  bool   `yaml:"behind_proxy"`
}

type Database struct {
	Dialect                  database.Dialect `yaml:"dialect"`
	DSN                      string           `yaml:"dsn"`
	SQLiteExternalWriteWatch bool             `yaml:"sqlite_external_write_watch"`
}

type Media struct {
	Dir                 string   `yaml:"dir"`
	MinFreeBytes        uint64   `yaml:"min_free_bytes"`
	StorageDriver       string   `yaml:"storage_driver"`
	S3Endpoint          string   `yaml:"s3_endpoint"`
	S3Bucket            string   `yaml:"s3_bucket"`
	S3Region            string   `yaml:"s3_region"`
	S3AccessKey         string   `yaml:"s3_access_key"`
	S3SecretKey         string   `yaml:"s3_secret_key"`
	S3UseSSL            bool     `yaml:"s3_use_ssl"`
	S3ForcePathStyle    bool     `yaml:"s3_force_path_style"`
	CDNBaseURL          string   `yaml:"cdn_base_url"`
	HotlinkProtection   bool     `yaml:"hotlink_protection"`
	AllowedRefererHosts []string `yaml:"allowed_referer_hosts"`
	WatermarkText       string   `yaml:"watermark_text"`
	WatermarkPosition   string   `yaml:"watermark_position"`
	WatermarkOpacity    float64  `yaml:"watermark_opacity"`
	WatermarkMargin     int      `yaml:"watermark_margin"`
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
	PreviewSigningKey   string   `yaml:"preview_signing_key" json:"-"`
	PasswordResetTTL    string   `yaml:"password_reset_ttl"`
	InvitationTTL       string   `yaml:"invitation_ttl"`
}

type I18N struct {
	Enabled       bool     `yaml:"enabled"`
	DefaultLocale string   `yaml:"default_locale"`
	Locales       []string `yaml:"locales"`
}

type Mail struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	From         string `yaml:"from"`
	DKIMSelector string `yaml:"dkim_selector"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	StartTLS     bool   `yaml:"starttls"`
	MaxPerSecond int    `yaml:"max_per_second"`
	MaxPerHour   int    `yaml:"max_per_hour"`
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

// GeoIP is opt-in and only used while the Forms module is active.
type GeoIP struct {
	DatabasePath      string   `yaml:"database_path"`
	TrustedProxyCIDRs []string `yaml:"trusted_proxy_cidrs"`
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
	SafeMode        *bool
}

func (c Config) PasswordResetDuration() time.Duration {
	value, _ := time.ParseDuration(c.Security.PasswordResetTTL)
	return value
}

func (c Config) InvitationDuration() time.Duration {
	value, _ := time.ParseDuration(c.Security.InvitationTTL)
	return value
}
