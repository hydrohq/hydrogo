package config

import (
	"regexp"
	"strings"
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
	ContentTypesDir string           `yaml:"content_types_dir"`
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
	Hooks           []SyncHook       `yaml:"hooks"`
	Backup          Backup           `yaml:"backup"`
	Audit           Audit            `yaml:"audit"`
	Limits          Limits           `yaml:"limits"`
	Commerce        Commerce         `yaml:"commerce"`
	Notifications   Notifications    `yaml:"notifications"`
}

// Notifications configures operational chat alerts (a new commerce order,
// a reconciliation drift finding) -- cross-cutting, not commerce-specific,
// so it lives at the top level rather than nested under Commerce, even
// though commerce is its first and only user today. Empty is valid: no
// project has to configure this to work, it just gets no alerts.
type Notifications struct {
	Telegram TelegramNotifications `yaml:"telegram"`
}

// TelegramNotifications holds one bot's credentials. BotToken is a secret:
// set it in a protected, untracked config file or the deployment
// environment, never in a committed preset.
type TelegramNotifications struct {
	BotToken string `yaml:"bot_token"`
	ChatID   string `yaml:"chat_id"`
}

// Enabled reports whether enough configuration is present to send.
func (t TelegramNotifications) Enabled() bool { return t.BotToken != "" && t.ChatID != "" }

// Commerce configures the optional payment providers modules/commerce can
// start a checkout against. Empty is a valid, fully functional
// configuration: Checkout and ConfirmPayment work without any provider
// configured, a project just has no way to redirect a shopper to a hosted
// payment page yet.
type Commerce struct {
	PayTR        PayTR        `yaml:"paytr"`
	Tax          Tax          `yaml:"tax"`
	Shipping     ShippingFee  `yaml:"shipping"`
	Currencies   Currencies   `yaml:"currencies"`
	YurticiKargo YurticiKargo `yaml:"yurtici_kargo"`
}

// Tax configures KDV/VAT for every checkout. The zero value (Rate 0) adds
// no tax at all -- a project that has not decided its tax setup yet is
// unaffected.
type Tax struct {
	// Rate is e.g. 0.20 for 20% KDV. Must be 0 (disabled) or in (0, 1).
	Rate float64 `yaml:"rate"`
	// Inclusive means catalog prices already include tax (common in
	// Turkish retail): the rate is then used only to break the tax amount
	// out for the receipt, never added on top of the price. False means
	// catalog prices are pre-tax and the rate is added at checkout.
	Inclusive bool `yaml:"inclusive"`
}

// ShippingFee configures a single flat shipping fee, optionally waived
// above a subtotal threshold. The zero value adds no shipping fee. Named
// ShippingFee, not Shipping, so it reads unambiguously next to
// commerce.ShippingProvider (a carrier integration -- Yurtiçi Kargo and
// friends) and the shipping_* columns on an order; this struct is pure
// pricing and has nothing to do with carriers. The yaml key stays
// `shipping:` -- only the Go type name changed.
type ShippingFee struct {
	FlatRateCents  int64 `yaml:"flat_rate_cents"`
	FreeAboveCents int64 `yaml:"free_above_cents"`
}

// Currencies configures the storefront's display-currency switcher. This
// never changes what is actually charged -- checkout and PayTR always use
// Default; Display/Rates are presentation only, a static, operator-
// maintained conversion a storefront can show a visitor without this
// project taking on live FX risk or multi-currency settlement. The zero
// value (no Display currencies) shows prices in Default only.
type Currencies struct {
	Default string             `yaml:"default"`
	Display []string           `yaml:"display"`
	Rates   map[string]float64 `yaml:"rates"`
}

// PayTR holds one merchant's PayTR credentials. MerchantKey and
// MerchantSalt are secrets: set them in a protected, untracked config file
// or through the deployment environment, never in a committed preset.
type PayTR struct {
	MerchantID     string `yaml:"merchant_id"`
	MerchantKey    string `yaml:"merchant_key"`
	MerchantSalt   string `yaml:"merchant_salt"`
	TestMode       bool   `yaml:"test_mode"`
	NoInstallment  bool   `yaml:"no_installment"`
	MaxInstallment int    `yaml:"max_installment"`
	OKURL          string `yaml:"ok_url"`
	FailURL        string `yaml:"fail_url"`
}

// Enabled reports whether enough PayTR configuration is present to start
// the provider; it is intentionally cheap to check from cmd/hydro's route
// wiring without importing modules/commerce/providers/paytr there.
func (p PayTR) Enabled() bool { return p.MerchantID != "" }

// YurticiKargo holds one account's Yurtiçi Kargo web service credentials --
// see modules/commerce/providers/yurticikargo's package doc comment for
// this integration's unverified status. WSUserName/WSPassword are secrets:
// set them in a protected, untracked config file or through the deployment
// environment, never in a committed preset.
type YurticiKargo struct {
	WSUserName string `yaml:"ws_username"`
	WSPassword string `yaml:"ws_password"`
	// Endpoint is the SOAP service URL. Empty means the publicly documented
	// one, which is plain HTTP; any http:// endpoint sends the credentials
	// unencrypted and therefore needs AllowInsecureEndpoint.
	Endpoint              string `yaml:"endpoint"`
	AllowInsecureEndpoint bool   `yaml:"allow_insecure_endpoint"`
}

// YurticiKargoDefaultEndpoint is the only endpoint Yurtiçi documents publicly.
const YurticiKargoDefaultEndpoint = "http://webservices.yurticikargo.com:8080/KOPSWebServices/ShippingOrderDispatcherServices"

// ResolvedEndpoint is Endpoint or the documented default.
func (y YurticiKargo) ResolvedEndpoint() string {
	if strings.TrimSpace(y.Endpoint) != "" {
		return strings.TrimSpace(y.Endpoint)
	}
	return YurticiKargoDefaultEndpoint
}

// Enabled reports whether enough configuration is present to start the
// provider, the same cheap pre-import check PayTR.Enabled gives cmd/hydro.
func (y YurticiKargo) Enabled() bool { return y.WSUserName != "" && y.WSPassword != "" }

// Limits holds operator-tunable resource caps.
type Limits struct {
	// RecordsPerType caps the records of one content type per tenant. Zero
	// keeps the built-in default; a negative value removes the cap.
	RecordsPerType int `yaml:"records_per_type"`
}

// Audit configures the optional detail recorded in the audit trail.
type Audit struct {
	// LogRequestBodies records the redacted, size-capped JSON body of every
	// state-changing admin API request, so the audit trail shows what was sent.
	LogRequestBodies bool `yaml:"log_request_bodies"`
	// Chain links every new audit event into a per-tenant hash chain so
	// tampering is detectable (hydro audit head | verify).
	Chain bool `yaml:"chain"`
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

// SyncHook is a side service the content types may consult before a change
// (see contenttypes hooks). The URL comes from the operator's config, never
// from a content type, so it may point at a service on the same host or
// network. SecretEnv names the environment variable holding the signing
// secret.
type SyncHook struct {
	ID        string `yaml:"id"`
	URL       string `yaml:"url"`
	SecretEnv string `yaml:"secret_env"`
	Secret    string `yaml:"-"`
	// Timeout caps one call (default 5s, at most 10s).
	Timeout string `yaml:"timeout"`
	// Schedules call the service on a clock: a signed request with topic
	// hook.schedule.<name> at every matching minute (UTC).
	Schedules []HookSchedule `yaml:"schedules"`
}

// HookSchedule fires the event Name at the five-field cron expression Cron.
type HookSchedule struct {
	Name string `yaml:"name" json:"name"`
	Cron string `yaml:"cron" json:"cron"`
}

type HTTP struct {
	Address      string `yaml:"address"`
	PublicOrigin string `yaml:"public_origin"`
	AutoTLS      bool   `yaml:"auto_tls"`
	Domain       string `yaml:"domain"`
	BehindProxy  bool   `yaml:"behind_proxy"`
	// TrustedProxies lists the proxy addresses (CIDR or bare IP) whose
	// X-Forwarded-For is believed. Empty with behind_proxy means a proxy on
	// the same host (loopback). Every other peer is taken at face value.
	TrustedProxies []string `yaml:"trusted_proxies"`
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
	// SkipDeliverySecrets lets commands that never call a hook or webhook
	// (migrate, backup, create-admin, ...) run without the secrets those
	// integrations read from the environment.
	SkipDeliverySecrets bool
}

func (c Config) PasswordResetDuration() time.Duration {
	value, _ := time.ParseDuration(c.Security.PasswordResetTTL)
	return value
}

func (c Config) InvitationDuration() time.Duration {
	value, _ := time.ParseDuration(c.Security.InvitationTTL)
	return value
}
