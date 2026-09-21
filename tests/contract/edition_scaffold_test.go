package contract

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

type generatedMigrationMetadata struct {
	Class struct {
		SQLite   string `yaml:"sqlite"`
		Postgres string `yaml:"postgres"`
	} `yaml:"class"`
	Transactional struct {
		SQLite   bool `yaml:"sqlite"`
		Postgres bool `yaml:"postgres"`
	} `yaml:"transactional"`
}

func TestCheckedInEditionMigrationsApplyToDisposableSQLite(t *testing.T) {
	for _, collection := range []string{"project", "article"} {
		dir := filepath.Join("..", "..", "examples", collection+"-edition", "generated", collection)
		sqlBytes, err := os.ReadFile(filepath.Join(dir, "migration.sql"))
		if err != nil {
			t.Fatal(err)
		}
		var metadata generatedMigrationMetadata
		metadataBytes, err := os.ReadFile(filepath.Join(dir, "migration.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		if err := yaml.Unmarshal(metadataBytes, &metadata); err != nil {
			t.Fatal(err)
		}
		if metadata.Class.SQLite != "expand" || metadata.Class.Postgres != "expand" || !metadata.Transactional.SQLite || !metadata.Transactional.Postgres {
			t.Fatalf("%s migration metadata is not dual-dialect additive: %+v", collection, metadata)
		}
		db, err := sql.Open("sqlite", ":memory:")
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.ExecContext(context.Background(), string(sqlBytes))
		_ = db.Close()
		if err != nil {
			t.Fatalf("%s generated migration failed on SQLite: %v", collection, err)
		}
		if strings.Contains(strings.ToUpper(string(sqlBytes)), "DROP TABLE") {
			t.Fatalf("%s generated migration contains destructive DDL", collection)
		}
	}
}
