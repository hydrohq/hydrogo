//go:build !hydro_compact

package database

import "testing"

func TestMySQLDriverIsIncludedInStandardBuild(t *testing.T) {
	if err := ValidateDialectAvailable(MySQL); err != nil {
		t.Fatalf("MySQL driver is unavailable in standard build: %v", err)
	}
	if placeholder(MySQL, 2) != "?" {
		t.Fatalf("MySQL placeholder = %q, want ?", placeholder(MySQL, 2))
	}
}
