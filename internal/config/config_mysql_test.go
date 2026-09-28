//go:build !hydro_compact

package config

import (
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func TestLoadAcceptsMySQLDialectInStandardBuild(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{
		"HYDRO_DATABASE_DIALECT": "mysql",
		"HYDRO_DATABASE_DSN":     "hydro:secret@tcp(127.0.0.1:3306)/hydro",
	}), Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Dialect != database.MySQL {
		t.Fatalf("database dialect = %q, want mysql", cfg.Database.Dialect)
	}
}
