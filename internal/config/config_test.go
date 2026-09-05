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
