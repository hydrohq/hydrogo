package config

import (
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

func TestLoadRejectsPartialSMTPConfiguration(t *testing.T) {
	_, err := Load("", lookup(map[string]string{"HYDRO_DATABASE_DSN": "file:test.sqlite", "HYDRO_SMTP_FROM": "from@example.test"}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), "mail host is required") {
		t.Fatalf("partial SMTP error = %v", err)
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
