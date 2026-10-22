package baselinegen_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/database/baselinegen"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

// The baseline is only trustworthy while a database it builds is the same as the
// one the chain builds. These tests compare the two for every module set that
// matters, on every dialect that is available: SQLite always, PostgreSQL and
// MySQL when HYDRO_TEST_POSTGRES_URL / HYDRO_TEST_MYSQL_URL name a server
// account that may create databases.

func providers(t *testing.T) []baselinegen.Provider {
	t.Helper()
	var list []baselinegen.Provider
	sqlite, err := baselinegen.NewProvider(database.SQLite, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	list = append(list, sqlite)
	for _, entry := range []struct {
		dialect database.Dialect
		env     string
	}{{database.Postgres, "HYDRO_TEST_POSTGRES_URL"}, {database.MySQL, "HYDRO_TEST_MYSQL_URL"}} {
		if dsn := os.Getenv(entry.env); dsn != "" {
			provider, err := baselinegen.NewProvider(entry.dialect, dsn, "")
			if err != nil {
				t.Fatal(err)
			}
			list = append(list, provider)
		}
	}
	return list
}

func moduleNames(t *testing.T) []string {
	t.Helper()
	groups, err := baselinegen.Groups(migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, group := range groups {
		for _, module := range group.Modules {
			seen[module] = true
		}
	}
	var names []string
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func enabledSet(modules []string) map[string]bool {
	if modules == nil {
		return nil
	}
	set := map[string]bool{}
	for _, module := range modules {
		set[module] = true
	}
	return set
}

func ledger(ctx context.Context, t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.QueryContext(ctx, "SELECT id, checksum FROM schema_migrations ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var id, checksum string
		if err := rows.Scan(&id, &checksum); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, id+" "+checksum)
	}
	return strings.Join(lines, "\n")
}

type state struct {
	snapshot *baselinegen.Snapshot
	ledger   string
}

// build applies each module set in turn to one fresh database, with or without
// the baseline, and returns the final state.
func build(ctx context.Context, t *testing.T, provider baselinegen.Provider, useBaseline bool, steps ...[]string) (*state, error) {
	t.Helper()
	db, cleanup, err := provider.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	return buildOn(ctx, t, db, provider.Dialect(), useBaseline, steps...)
}

func buildOn(ctx context.Context, t *testing.T, db *sql.DB, dialect database.Dialect, useBaseline bool, steps ...[]string) (*state, error) {
	t.Helper()
	for _, modules := range steps {
		var source = baselinegen.ChainOnly(migrations.FS)
		if useBaseline {
			source = migrations.FS
		}
		if err := database.ApplyMigrationsForModules(ctx, db, dialect, source, enabledSet(modules)); err != nil {
			return nil, err
		}
	}
	snapshot, err := baselinegen.Take(ctx, db, dialect)
	if err != nil {
		t.Fatal(err)
	}
	return &state{snapshot: snapshot, ledger: ledger(ctx, t, db)}, nil
}

func requireSame(t *testing.T, label string, want, got *state) {
	t.Helper()
	if problems := baselinegen.Diff(want.snapshot, got.snapshot); len(problems) > 0 {
		t.Fatalf("%s: baseline and chain databases differ:\n%s", label, strings.Join(problems, "\n"))
	}
	if want.ledger != got.ledger {
		t.Fatalf("%s: the migration ledgers differ:\nchain:\n%s\nbaseline:\n%s", label, want.ledger, got.ledger)
	}
}

// valid reports whether the chain alone can build the module set; some sets are
// not configurations the product supports (forms needs site's table).
func valid(ctx context.Context, t *testing.T, provider baselinegen.Provider, steps ...[]string) (*state, bool) {
	t.Helper()
	chain, err := build(ctx, t, provider, false, steps...)
	if err != nil {
		return nil, false
	}
	return chain, true
}

func TestBaselineBuildsTheSameDatabaseAsTheChain(t *testing.T) {
	ctx := context.Background()
	names := moduleNames(t)
	presets := map[string][]string{
		"core":      {},
		"crm":       {"auth", "media", "collections"},
		"news":      {"auth", "members", "pages", "media", "site", "news", "collections", "analytics"},
		"all":       nil,
		"commerce":  {"pages", "media", "commerce", "members"},
		"sectoral":  {"pages", "site", "media", "team", "partners", "faq"},
		"pages+all": names,
	}
	for _, provider := range providers(t) {
		provider := provider
		t.Run(string(provider.Dialect()), func(t *testing.T) {
			for name, modules := range presets {
				chain, ok := valid(ctx, t, provider, modules)
				if !ok {
					t.Logf("preset %s is not a valid configuration for the chain; skipped", name)
					continue
				}
				base, err := build(ctx, t, provider, true, modules)
				if err != nil {
					t.Fatalf("preset %s: baseline build failed: %v", name, err)
				}
				requireSame(t, "preset "+name, chain, base)
			}
			for _, module := range names {
				chain, ok := valid(ctx, t, provider, []string{module})
				if !ok {
					t.Logf("module %s alone is not a valid configuration for the chain; skipped", module)
					continue
				}
				base, err := build(ctx, t, provider, true, []string{module})
				if err != nil {
					t.Fatalf("module %s: baseline build failed: %v", module, err)
				}
				requireSame(t, "module "+module, chain, base)
			}
		})
	}
}

// The fix ADR 0009 is about: a database created without a module can enable it
// later, whether the database came from the baseline or from the old chain.
func TestEnablingAModuleLaterMatchesEnablingItFromTheStart(t *testing.T) {
	ctx := context.Background()
	names := moduleNames(t)
	for _, provider := range providers(t) {
		provider := provider
		t.Run(string(provider.Dialect()), func(t *testing.T) {
			for _, module := range names {
				want, ok := valid(ctx, t, provider, []string{module})
				if !ok {
					continue
				}
				fromBaseline, err := build(ctx, t, provider, true, []string{}, []string{module})
				if err != nil {
					t.Fatalf("enable %s on a baseline-built core database: %v", module, err)
				}
				requireSame(t, "baseline core then "+module, want, fromBaseline)

				legacy, err := buildLegacyThenBaseline(ctx, t, provider, []string{}, []string{module})
				if err != nil {
					t.Fatalf("enable %s on a chain-built core database: %v", module, err)
				}
				requireSame(t, "chain core then "+module, want, legacy)
			}
		})
	}
}

// buildLegacyThenBaseline creates the database with the chain only (what existing
// installations look like) and then runs the current binary, baseline included.
func buildLegacyThenBaseline(ctx context.Context, t *testing.T, provider baselinegen.Provider, first, second []string) (*state, error) {
	t.Helper()
	db, cleanup, err := provider.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if _, err := buildOn(ctx, t, db, provider.Dialect(), false, first); err != nil {
		return nil, fmt.Errorf("legacy chain: %w", err)
	}
	return buildOn(ctx, t, db, provider.Dialect(), true, second)
}

func TestEnablingTwoModulesInEitherOrderMatchesTheChain(t *testing.T) {
	if testing.Short() {
		t.Skip("compares every module pair; the full run is part of go test without -short")
	}
	ctx := context.Background()
	names := moduleNames(t)
	for _, provider := range providers(t) {
		provider := provider
		t.Run(string(provider.Dialect()), func(t *testing.T) {
			pairs := 0
			for _, first := range names {
				for _, second := range names {
					if first >= second {
						continue
					}
					if provider.Dialect() != database.SQLite && !samplePair(first, second) {
						continue
					}
					want, ok := valid(ctx, t, provider, []string{first, second})
					if !ok {
						continue
					}
					pairs++
					for _, order := range [][]string{{first, second}, {second, first}} {
						got, err := build(ctx, t, provider, true, []string{}, []string{order[0]}, []string{first, second})
						if err != nil {
							// the first module alone may be unsupported (forms before site)
							continue
						}
						requireSame(t, fmt.Sprintf("core, %s, then %s", order[0], order[1]), want, got)
					}
				}
			}
			if pairs == 0 {
				t.Fatal("no module pair was compared")
			}
		})
	}
}

// samplePair keeps the slower server dialects to pairs that exercise every kind
// of interaction: shared tables, combination groups and audit/outbox ownership.
func samplePair(a, b string) bool {
	key := a + "+" + b
	switch key {
	case "media+pages", "news+pages", "members+pages", "pages+site", "analytics+pages", "auth+members", "commerce+pages", "media+news":
		return true
	}
	return false
}

func TestRunningTheMigrationsAgainChangesNothing(t *testing.T) {
	ctx := context.Background()
	for _, provider := range providers(t) {
		provider := provider
		t.Run(string(provider.Dialect()), func(t *testing.T) {
			db, cleanup, err := provider.New(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()
			first, err := buildOn(ctx, t, db, provider.Dialect(), true, nil)
			if err != nil {
				t.Fatal(err)
			}
			second, err := buildOn(ctx, t, db, provider.Dialect(), true, nil)
			if err != nil {
				t.Fatalf("second run: %v", err)
			}
			requireSame(t, "second run", first, second)
		})
	}
}

func databaseGroupIDs() ([]string, error) { return database.BaselineGroupIDs(migrations.FS) }
