package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadPrecedenceAndSecretConflict(t *testing.T) {
	path := writeConfig(t, `http:
  address: "127.0.0.1:7000"
database:
  dialect: "sqlite"
`)
	env := map[string]string{
		"HYDRO_HTTP_ADDRESS":     "127.0.0.1:7001",
		"HYDRO_DATABASE_DIALECT": "postgres",
		"HYDRO_DATABASE_DSN":     "postgres://from-env",
	}
	address := "127.0.0.1:7002"
	dialect := "sqlite"
	cfg, err := Load(path, lookup(env), Overrides{HTTPAddress: &address, DatabaseDialect: &dialect})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Address != address || string(cfg.Database.Dialect) != dialect || cfg.Database.DSN != env["HYDRO_DATABASE_DSN"] {
		t.Fatalf("unexpected config: %+v", cfg)
	}

	dsn := "file:from-flag.sqlite"
	_, err = Load(path, lookup(env), Overrides{DatabaseDSN: &dsn})
	if err == nil || !strings.Contains(err.Error(), "multiple secret sources") {
		t.Fatalf("secret conflict error = %v", err)
	}
}

func TestSafeModeConfigPrecedenceAndValidation(t *testing.T) {
	path := writeConfig(t, "safe_mode: true\n")
	cfg, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:safe.sqlite", "HYDRO_SAFE_MODE": "false"}), Overrides{})
	if err != nil || cfg.SafeMode {
		t.Fatalf("environment safe mode = %v, error = %v", cfg.SafeMode, err)
	}
	enabled := true
	cfg, err = Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:safe.sqlite", "HYDRO_SAFE_MODE": "false"}), Overrides{SafeMode: &enabled})
	if err != nil || !cfg.SafeMode {
		t.Fatalf("flag safe mode = %v, error = %v", cfg.SafeMode, err)
	}
	if _, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:safe.sqlite", "HYDRO_SAFE_MODE": "enabled"}), Overrides{}); err == nil || !strings.Contains(err.Error(), "HYDRO_SAFE_MODE") {
		t.Fatalf("invalid safe mode error = %v", err)
	}
}

func TestSecurityTokenLifetimeChoices(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:token-ttl.sqlite", "HYDRO_PASSWORD_RESET_TTL": "24h", "HYDRO_INVITATION_TTL": "168h"}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PasswordResetDuration().Hours() != 24 || cfg.InvitationDuration().Hours() != 168 {
		t.Fatalf("unexpected TTLs: reset=%s invite=%s", cfg.PasswordResetDuration(), cfg.InvitationDuration())
	}
	for _, env := range []map[string]string{
		{"HYDRO_DATABASE_DSN": "file:bad-reset.sqlite", "HYDRO_PASSWORD_RESET_TTL": "2h"},
		{"HYDRO_DATABASE_DSN": "file:bad-invite.sqlite", "HYDRO_INVITATION_TTL": "48h"},
	} {
		if _, err := Load("", lookup(env), Overrides{}); err == nil {
			t.Errorf("unsupported token lifetime accepted: %#v", env)
		}
	}
}

func TestLoadSentryDSNFromEnvironment(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:sentry.sqlite", "HYDRO_SENTRY_DSN": "https://public@example.test/7"}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Observability.SentryDSN != "https://public@example.test/7" {
		t.Fatalf("Sentry DSN = %q", cfg.Observability.SentryDSN)
	}
	if _, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:sentry.sqlite", "HYDRO_SENTRY_DSN": strings.Repeat("x", 4097)}), Overrides{}); err == nil || !strings.Contains(err.Error(), "Sentry") {
		t.Fatalf("oversized Sentry DSN error = %v", err)
	}
}

func TestBackupSigningSeedIsEnvironmentOnlyAndValidated(t *testing.T) {
	seed := strings.Repeat("ab", 32)
	cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:backup-signing.sqlite", "HYDRO_BACKUP_SIGNING_SEED": seed}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Backup.SigningSeed != seed {
		t.Fatal("backup signing seed was not loaded from its environment source")
	}
	if _, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:backup-signing-bad.sqlite", "HYDRO_BACKUP_SIGNING_SEED": "not-a-seed"}), Overrides{}); err == nil {
		t.Fatal("invalid backup signing seed accepted")
	}
	encoded, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), seed) {
		t.Fatal("backup signing seed leaked through config JSON")
	}
}

func TestLoadNormalizesModuleNamesBeforeRuntimeComposition(t *testing.T) {
	path := writeConfig(t, "modules: [ Core, ' pages ', MEDIA ]\n")
	cfg, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"core", "pages", "media"}
	if !reflect.DeepEqual(cfg.Modules, want) {
		t.Fatalf("normalized modules = %#v, want %#v", cfg.Modules, want)
	}
}

func TestLoadRejectsInvalidOrDuplicateLocales(t *testing.T) {
	for _, locales := range []string{`["tr", "TR"]`, `["tr", "xx_xx"]`} {
		path := writeConfig(t, "i18n:\n  enabled: true\n  default_locale: tr\n  locales: "+locales+"\n")
		if _, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}), Overrides{}); err == nil || !strings.Contains(err.Error(), "i18n") {
			t.Fatalf("locales %s accepted/error=%v", locales, err)
		}
	}
}

func TestLoadRejectsDefaultLocaleOutsideEnabledList(t *testing.T) {
	path := writeConfig(t, "i18n:\n  enabled: true\n  default_locale: en\n  locales: [tr]\n")
	if _, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}), Overrides{}); err == nil || !strings.Contains(err.Error(), "not enabled") {
		t.Fatalf("default locale mismatch error = %v", err)
	}
}

func TestLoadRejectsUnknownKey(t *testing.T) {
	path := writeConfig(t, "unknown: true\n")
	_, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), "field unknown not found") {
		t.Fatalf("unknown key error = %v", err)
	}
}

func TestLoadCapacityProfile(t *testing.T) {
	path := writeConfig(t, "profile: standard\ndatabase:\n  dsn: postgres://example\n")
	cfg, err := Load(path, lookup(nil), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if string(cfg.Profile) != "standard" {
		t.Fatalf("profile = %q", cfg.Profile)
	}
	_, err = Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite", "HYDRO_CAPACITY_PROFILE": "invalid"}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), "unsupported capacity profile") {
		t.Fatalf("invalid profile error = %v", err)
	}
}

func TestLoadTreatsNilEnvironmentLookupAsEmpty(t *testing.T) {
	dsn := "file:nil-lookup.sqlite"
	cfg, err := Load("", nil, Overrides{DatabaseDSN: &dsn})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.DSN != dsn {
		t.Fatalf("database DSN = %q, want %q", cfg.Database.DSN, dsn)
	}
}

func TestAutoTLSRequiresDomainAndRejectsReverseProxy(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	cfg.HTTP = HTTP{Address: "127.0.0.1:8080", AutoTLS: true}
	if err := cfg.Validate(); err == nil {
		t.Fatal("AutoTLS without a domain accepted")
	}
	cfg.HTTP = HTTP{Address: "127.0.0.1:8080", AutoTLS: true, Domain: "site.example", BehindProxy: true}
	if err := cfg.Validate(); err == nil {
		t.Fatal("AutoTLS behind proxy accepted")
	}
	cfg.HTTP = HTTP{Address: "127.0.0.1:8080", AutoTLS: true, Domain: "site.example"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid AutoTLS config: %v", err)
	}
}

func TestNoCookieAnalyticsRequiresExplicitEnablementAndSharedKey(t *testing.T) {
	if cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:analytics-disabled.sqlite"}), Overrides{}); err != nil || cfg.Analytics.NoCookieEnabled {
		t.Fatalf("default analytics config=%+v err=%v", cfg.Analytics, err)
	}
	if _, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:analytics-bad.sqlite", "HYDRO_ANALYTICS_NO_COOKIE_ENABLED": "true"}), Overrides{}); err == nil || !strings.Contains(err.Error(), "analytics module") {
		t.Fatalf("enabled analytics without shared key error=%v", err)
	}
	key := strings.Repeat("k", 32)
	path := writeConfig(t, "modules: [core, analytics]\n")
	_, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:analytics-bad-key.sqlite", "HYDRO_ANALYTICS_NO_COOKIE_ENABLED": "true", "HYDRO_ANALYTICS_HASH_KEY": "short"}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), "at least 32 bytes") {
		t.Fatalf("enabled analytics without sufficient key error=%v", err)
	}
	cfg, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:analytics-enabled.sqlite", "HYDRO_ANALYTICS_NO_COOKIE_ENABLED": "true", "HYDRO_ANALYTICS_HASH_KEY": key}), Overrides{})
	if err != nil || !cfg.Analytics.NoCookieEnabled || cfg.Analytics.HashKey != key {
		t.Fatalf("enabled analytics config=%+v err=%v", cfg.Analytics, err)
	}
}

func TestTelegramNotificationsRequireBothBotTokenAndChatID(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:telegram.sqlite"}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	cfg.Notifications.Telegram = TelegramNotifications{BotToken: "token-only"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("bot token without chat id accepted")
	}
	cfg.Notifications.Telegram = TelegramNotifications{ChatID: "chat-only"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("chat id without bot token accepted")
	}
	cfg.Notifications.Telegram = TelegramNotifications{BotToken: "token", ChatID: "chat"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid telegram config: %v", err)
	}
	cfg.Notifications.Telegram = TelegramNotifications{}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unconfigured telegram must stay valid: %v", err)
	}
}

func TestPayTRConfigurationRequiresKeyAndSaltWhenEnabled(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:paytr.sqlite"}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	cfg.Commerce.PayTR = PayTR{MerchantID: "merchant-1"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("PayTR merchant id without key and salt accepted")
	}
	cfg.Commerce.PayTR = PayTR{MerchantID: "merchant-1", MerchantKey: "key", MerchantSalt: "salt"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid PayTR config: %v", err)
	}
	cfg.Commerce.PayTR = PayTR{}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("PayTR left unconfigured must stay valid: %v", err)
	}
}

func TestTaxRateMustBeAFractionBelowOne(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:tax.sqlite"}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	cfg.Commerce.Tax = Tax{Rate: 0}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("disabled tax (rate 0) must stay valid: %v", err)
	}
	cfg.Commerce.Tax = Tax{Rate: 0.20}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid tax rate: %v", err)
	}
	cfg.Commerce.Tax = Tax{Rate: -0.1}
	if err := cfg.Validate(); err == nil {
		t.Fatal("negative tax rate accepted")
	}
	cfg.Commerce.Tax = Tax{Rate: 1}
	if err := cfg.Validate(); err == nil {
		t.Fatal("100%+ tax rate accepted")
	}
}

func TestShippingRatesMustBeNonNegative(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:shipping.sqlite"}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	cfg.Commerce.Shipping = Shipping{FlatRateCents: -100}
	if err := cfg.Validate(); err == nil {
		t.Fatal("negative flat_rate_cents accepted")
	}
	cfg.Commerce.Shipping = Shipping{FlatRateCents: 500, FreeAboveCents: -1}
	if err := cfg.Validate(); err == nil {
		t.Fatal("negative free_above_cents accepted")
	}
	cfg.Commerce.Shipping = Shipping{FlatRateCents: 500, FreeAboveCents: 10000}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid shipping config: %v", err)
	}
}

func TestDisplayCurrenciesRequireARateAndADefault(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:currencies.sqlite"}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	cfg.Commerce.Currencies = Currencies{Display: []string{"USD"}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("a display currency with no default and no rate accepted")
	}
	cfg.Commerce.Currencies = Currencies{Default: "TRY", Display: []string{"USD"}, Rates: map[string]float64{}}
	if err := cfg.Validate(); err == nil {
		t.Fatal("a display currency with no rate accepted")
	}
	cfg.Commerce.Currencies = Currencies{Default: "TRY", Display: []string{"USD"}, Rates: map[string]float64{"USD": 0.03}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid currency config: %v", err)
	}
	cfg.Commerce.Currencies = Currencies{}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("no display currencies must stay valid: %v", err)
	}
}

func TestGeoIPConfigurationBoundsTrustedProxyRanges(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:geoip.sqlite"}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	cfg.GeoIP = GeoIP{TrustedProxyCIDRs: []string{"192.0.2.0/24"}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid GeoIP config: %v", err)
	}
	cfg.GeoIP.TrustedProxyCIDRs = []string{"not-a-cidr"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("invalid trusted proxy CIDR accepted")
	}
	cfg.GeoIP = GeoIP{DatabasePath: strings.Repeat("x", 4097)}
	if err := cfg.Validate(); err == nil {
		t.Fatal("unbounded GeoIP database path accepted")
	}
}

func TestLoadWebhookRequiresPublishedTopicAndExplicitPIIOptIn(t *testing.T) {
	base := "webhooks:\n  - id: consumer\n    url: https://consumer.example.test/hooks\n    topics: [%s]\n"
	for _, topic := range []string{"unknown.event", "user.invited"} {
		path := writeConfig(t, fmt.Sprintf(base, topic))
		_, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite", "HYDRO_WEBHOOK_SECRET": "test-secret"}), Overrides{})
		if err == nil {
			t.Fatalf("webhook topic %q accepted without policy", topic)
		}
	}
	path := writeConfig(t, "webhooks:\n  - id: consumer\n    url: https://consumer.example.test/hooks\n    allow_pii: true\n    topics: [user.invited]\n")
	cfg, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite", "HYDRO_WEBHOOK_SECRET": "test-secret"}), Overrides{})
	if err != nil || !cfg.Webhooks[0].AllowPII {
		t.Fatalf("PII webhook opt-in = %#v, err=%v", cfg.Webhooks, err)
	}
}

func TestLoadEnablesOptInDirectMXWithSenderAddressOnly(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite", "HYDRO_SMTP_FROM": "contact@example.test"}), Overrides{})
	if err != nil || cfg.Mail.Host != "" || cfg.Mail.From != "contact@example.test" {
		t.Fatalf("direct MX config = %#v, error = %v", cfg.Mail, err)
	}
}

func TestLoadSMTPRateLimitsFromEnvironment(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite", "HYDRO_SMTP_MAX_PER_SECOND": "3", "HYDRO_SMTP_MAX_PER_HOUR": "42"}), Overrides{})
	if err != nil || cfg.Mail.MaxPerSecond != 3 || cfg.Mail.MaxPerHour != 42 {
		t.Fatalf("SMTP rate config = %#v, error = %v", cfg.Mail, err)
	}
	_, err = Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite", "HYDRO_SMTP_MAX_PER_SECOND": "-1"}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), "SMTP rate limits") {
		t.Fatalf("invalid SMTP rate config error = %v", err)
	}
}

func TestLoadRejectsInvalidDirectMXSenderAddress(t *testing.T) {
	_, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite", "HYDRO_SMTP_FROM": "bad\r\nBcc: victim@example.test"}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), "mail from is invalid") {
		t.Fatalf("invalid sender error = %v", err)
	}
}

func TestLoadTurnstileConfiguration(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":               "file:test.sqlite",
		"HYDRO_FORMS_TURNSTILE_SITE_KEY":   "site-key",
		"HYDRO_FORMS_TURNSTILE_SECRET_KEY": "secret-key",
	}), Overrides{})
	if err != nil || cfg.Forms.TurnstileSiteKey != "site-key" || cfg.Forms.TurnstileSecretKey != "secret-key" {
		t.Fatalf("Turnstile config = %#v, error = %v", cfg.Forms, err)
	}
	_, err = Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":             "file:test.sqlite",
		"HYDRO_FORMS_TURNSTILE_SITE_KEY": "site-key",
	}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), "configured together") {
		t.Fatalf("partial Turnstile config error = %v", err)
	}
}

func TestLoadSEOAssetDirectories(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite", "HYDRO_SEO_OG_CACHE_DIR": "data/og", "HYDRO_SEO_FONTS_DIR": "public/fonts"}), Overrides{})
	if err != nil || cfg.SEO.OGCacheDir != "data/og" || cfg.SEO.FontsDir != "public/fonts" {
		t.Fatalf("SEO config=%#v err=%v", cfg.SEO, err)
	}
	_, err = Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite", "HYDRO_SEO_FONTS_DIR": "fonts\nmalicious"}), Overrides{})
	if err == nil {
		t.Fatal("newline in font directory was accepted")
	}
}

func TestLoadRejectsSMTPAuthWithoutTLS(t *testing.T) {
	_, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":  "file:test.sqlite",
		"HYDRO_SMTP_HOST":     "smtp.example.test",
		"HYDRO_SMTP_FROM":     "from@example.test",
		"HYDRO_SMTP_USERNAME": "user",
		"HYDRO_SMTP_STARTTLS": "false",
	}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), "STARTTLS") {
		t.Fatalf("SMTP AUTH without TLS error = %v", err)
	}
}

func TestLoadSecurityTransportOptions(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":   "file:test.sqlite",
		"HYDRO_SECURE_COOKIES": "true",
		"HYDRO_ENABLE_HSTS":    "true",
		"HYDRO_PUBLIC_ORIGIN":  "https://example.test",
	}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Security.SecureCookies || !cfg.Security.EnableHSTS {
		t.Fatalf("security transport options were not loaded: %+v", cfg.Security)
	}
	if _, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":   "file:test.sqlite",
		"HYDRO_SECURE_COOKIES": "maybe",
	}), Overrides{}); err == nil || !strings.Contains(err.Error(), "HYDRO_SECURE_COOKIES") {
		t.Fatalf("invalid secure cookie option error = %v", err)
	}
	if _, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":  "file:test.sqlite",
		"HYDRO_ENABLE_HSTS":   "true",
		"HYDRO_PUBLIC_ORIGIN": "http://example.test",
	}), Overrides{}); err == nil || !strings.Contains(err.Error(), "HSTS") {
		t.Fatalf("insecure HSTS configuration error = %v", err)
	}
	if _, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":  "file:test.sqlite",
		"HYDRO_ENABLE_HSTS":   "true",
		"HYDRO_PUBLIC_ORIGIN": "https://example.test",
	}), Overrides{}); err == nil || !strings.Contains(err.Error(), "secure cookies") {
		t.Fatalf("incomplete HSTS configuration error = %v", err)
	}
	if _, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":   "file:test.sqlite",
		"HYDRO_SECURE_COOKIES": "true",
	}), Overrides{}); err == nil || !strings.Contains(err.Error(), "secure cookies") {
		t.Fatalf("secure cookie without HTTPS origin error = %v", err)
	}
	if _, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":  "file:test.sqlite",
		"HYDRO_PUBLIC_ORIGIN": "https://example.test",
	}), Overrides{}); err == nil || !strings.Contains(err.Error(), "requires secure cookies") {
		t.Fatalf("HTTPS origin without secure cookies error = %v", err)
	}
}

func TestLoadSQLiteExternalWriteWatch(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":                "file:test.sqlite",
		"HYDRO_SQLITE_EXTERNAL_WRITE_WATCH": "true",
	}), Overrides{})
	if err != nil || !cfg.Database.SQLiteExternalWriteWatch {
		t.Fatalf("sqlite external-write watch = %v, err=%v", cfg.Database.SQLiteExternalWriteWatch, err)
	}
	if _, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":                "file:test.sqlite",
		"HYDRO_SQLITE_EXTERNAL_WRITE_WATCH": "maybe",
	}), Overrides{}); err == nil || !strings.Contains(err.Error(), "HYDRO_SQLITE_EXTERNAL_WRITE_WATCH") {
		t.Fatalf("invalid sqlite external-write watch error = %v", err)
	}
}

func TestLoadTenancyHostMap(t *testing.T) {
	path := writeConfig(t, `tenancy:
  enabled: true
  hosts:
    alpha.example.test: tenant-a
`)
	cfg, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}), Overrides{})
	if err != nil || !cfg.Tenancy.Enabled || !cfg.Tenancy.RequireHost || cfg.Tenancy.Hosts["alpha.example.test"] != "tenant-a" {
		t.Fatalf("tenancy config = %#v err=%v", cfg.Tenancy, err)
	}
	path = writeConfig(t, `tenancy:
  enabled: true
  hosts:
    "bad host": tenant-a
`)
	if _, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}), Overrides{}); err == nil || !strings.Contains(err.Error(), "tenancy") {
		t.Fatalf("invalid tenancy host accepted: %v", err)
	}
	path = writeConfig(t, `tenancy:
  enabled: true
  hosts:
    "alpha.example.test:8443": tenant-a
`)
	if _, err := Load(path, lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}), Overrides{}); err == nil || !strings.Contains(err.Error(), "tenancy") {
		t.Fatalf("tenancy host with port accepted: %v", err)
	}
}

func TestValidateRejectsNonOriginPublicURLs(t *testing.T) {
	for _, value := range []string{
		"https://user:pass@example.test",
		"https://example.test/base",
		"https://example.test/?tracking=1",
		"https://example.test/#fragment",
	} {
		_, err := Load("", lookup(map[string]string{
			"HYDRO_DATABASE_DSN":  "file:test.sqlite",
			"HYDRO_PUBLIC_ORIGIN": value,
		}), Overrides{})
		if err == nil || !strings.Contains(err.Error(), "invalid public origin") {
			t.Errorf("public origin %q error = %v", value, err)
		}
	}
}

func TestLoadValidatesPublicAPIOriginAllowlist(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":       "file:test.sqlite",
		"HYDRO_PUBLIC_API_ORIGINS": "https://store.example.test,http://localhost:3000",
	}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Security.PublicAPIOrigins) != 2 {
		t.Fatalf("public API origins = %#v", cfg.Security.PublicAPIOrigins)
	}
	_, err = Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":       "file:test.sqlite",
		"HYDRO_PUBLIC_API_ORIGINS": "https://store.example.test/path",
	}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), "invalid public API origin") {
		t.Fatalf("invalid API origin error = %v", err)
	}
}

func TestLoadI18NConfig(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DSN":   "file:test.sqlite",
		"HYDRO_I18N_ENABLED":   "true",
		"HYDRO_DEFAULT_LOCALE": "en",
		"HYDRO_LOCALES":        "en,tr,de",
	}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.I18N.Enabled || cfg.I18N.DefaultLocale != "en" || len(cfg.I18N.Locales) != 3 {
		t.Fatalf("unexpected I18N config: %+v", cfg.I18N)
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hydro.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func lookup(values map[string]string) LookupEnv {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func TestLoadSyncHooksNeedASecretFromTheEnvironment(t *testing.T) {
	body := "hooks:\n  - id: ledger\n    url: http://127.0.0.1:9100/hydro\n    secret_env: LEDGER_SECRET\n    timeout: 3s\n"
	env := map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}
	if _, err := Load(writeConfig(t, body), lookup(env), Overrides{}); err == nil {
		t.Fatal("hook without a secret accepted")
	}
	env["LEDGER_SECRET"] = "0123456789abcdef"
	cfg, err := Load(writeConfig(t, body), lookup(env), Overrides{})
	if err != nil || cfg.Hooks[0].Secret != "0123456789abcdef" {
		t.Fatalf("hooks = %#v err=%v", cfg.Hooks, err)
	}
	if _, err := Load(writeConfig(t, strings.Replace(body, "3s", "1m", 1)), lookup(env), Overrides{}); err == nil {
		t.Fatal("hook timeout above 10s accepted")
	}
}

func TestMaintenanceCommandsLoadWithoutDeliverySecrets(t *testing.T) {
	body := "hooks:\n  - id: ledger\n    url: http://127.0.0.1:9100/hydro\n    secret_env: LEDGER_SECRET\n"
	env := map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite"}
	if _, err := Load(writeConfig(t, body), lookup(env), Overrides{SkipDeliverySecrets: true}); err != nil {
		t.Fatalf("a command that never calls the hook must not need its secret: %v", err)
	}
	// Everything else is still checked.
	bad := strings.Replace(body, "http://127.0.0.1:9100/hydro", "not a url", 1)
	if _, err := Load(writeConfig(t, bad), lookup(env), Overrides{SkipDeliverySecrets: true}); err == nil {
		t.Fatal("an invalid hook URL was accepted")
	}
	// A real secret is kept, not replaced by the placeholder.
	env["LEDGER_SECRET"] = "0123456789abcdef"
	cfg, err := Load(writeConfig(t, body), lookup(env), Overrides{SkipDeliverySecrets: true})
	if err != nil || cfg.Hooks[0].Secret != "0123456789abcdef" {
		t.Fatalf("hooks = %#v err=%v", cfg.Hooks, err)
	}
}

func TestLoadHookSchedulesNeedValidCronExpressions(t *testing.T) {
	env := map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite", "LEDGER_SECRET": "0123456789abcdef"}
	head := "hooks:\n  - id: ledger\n    url: http://127.0.0.1:9100/hydro\n    secret_env: LEDGER_SECRET\n    schedules:\n"
	if cfg, err := Load(writeConfig(t, head+"      - {name: nightly, cron: \"0 2 * * *\"}\n"), lookup(env), Overrides{}); err != nil || len(cfg.Hooks[0].Schedules) != 1 {
		t.Fatalf("valid schedule: %v %#v", err, cfg.Hooks)
	}
	for _, bad := range []string{"      - {name: nightly, cron: \"61 2 * * *\"}\n", "      - {name: Bad Name, cron: \"0 2 * * *\"}\n", "      - {name: a, cron: \"0 2 * * *\"}\n      - {name: a, cron: \"0 3 * * *\"}\n"} {
		if _, err := Load(writeConfig(t, head+bad), lookup(env), Overrides{}); err == nil {
			t.Fatalf("accepted: %s", bad)
		}
	}
}
