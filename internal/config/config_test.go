package config

import (
	"os"
	"path/filepath"
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
