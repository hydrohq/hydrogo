//go:build hydro_compact

package database

import (
	"context"
	"strings"
	"testing"
)

func TestCompactBuildExcludesPostgresDriverAndFailsClosed(t *testing.T) {
	if !driverRegistered("sqlite") {
		t.Fatal("Compact build is missing the SQLite driver")
	}
	if driverRegistered("pgx") {
		t.Fatal("Compact build unexpectedly registered the PostgreSQL driver")
	}
	if _, err := Open(context.Background(), Postgres, "postgres://unused"); err == nil || !strings.Contains(err.Error(), `driver "pgx" is not included`) {
		t.Fatalf("PostgreSQL selection did not fail with an explicit build-capability error: %v", err)
	}
}
