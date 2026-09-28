//go:build hydro_compact

package config

import (
	"strings"
	"testing"
)

func TestCompactConfigRejectsUnlinkedPostgresDriver(t *testing.T) {
	_, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DIALECT": "postgres",
		"HYDRO_DATABASE_DSN":     "postgres://unused",
	}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), `driver "pgx" is not included`) {
		t.Fatalf("Compact config accepted unlinked PostgreSQL driver: %v", err)
	}
}

func TestCompactConfigRejectsUnlinkedMySQLDriver(t *testing.T) {
	_, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DIALECT": "mysql",
		"HYDRO_DATABASE_DSN":     "hydro:unused@/unused",
	}), Overrides{})
	if err == nil || !strings.Contains(err.Error(), `driver "mysql" is not included`) {
		t.Fatalf("Compact config accepted unlinked MySQL driver: %v", err)
	}
}
