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
	if driverRegistered("mysql") {
		t.Fatal("Compact build unexpectedly registered the MySQL driver")
	}
	if _, err := Open(context.Background(), Postgres, "postgres://unused"); err == nil || !strings.Contains(err.Error(), `driver "pgx" is not included`) {
		t.Fatalf("PostgreSQL selection did not fail with an explicit build-capability error: %v", err)
	}
	if _, err := Open(context.Background(), MySQL, "user:password@/unused"); err == nil || !strings.Contains(err.Error(), `driver "mysql" is not included`) {
		t.Fatalf("MySQL selection did not fail with an explicit build-capability error: %v", err)
	}
}
