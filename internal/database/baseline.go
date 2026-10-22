package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"gopkg.in/yaml.v3"
)

// A baseline is the generated end state of a group of migrations (ADR 0009):
// one schema script per dialect that stands in for the chain on a database that
// has applied none of the group's migrations. The ledger afterwards reads exactly
// as if the chain had run, so later migrations, checksum checks and downgrades of
// the binary behave the same.
//
// Groups partition the chain by the modules a migration needs:
//   - the core group (no modules) holds every migration that always applies;
//   - a module group holds the migrations of that module, including core
//     migrations that require only that module;
//   - a combination group holds migrations that require several modules together.
//
// A group is applied only while all of its migrations are pending, and only on a
// database whose ledger is group-consistent: every group is either fully applied
// or fully pending. A database that ran part of a group (an older release in the
// middle of its upgrade) has objects the baseline would create again, so it keeps
// running the chain for everything that is left.

const (
	// manifest.yml, not .yaml: the migrations embed pattern */*.yaml must not
	// pick it up, or "baseline" would list as a migration directory.
	baselineManifestPath = "manifest.yml"
	baselineVersion      = 1
)

// BaselineProvider is implemented by migration bundles that carry a baseline.
type BaselineProvider interface {
	BaselineFS() fs.FS
}

type baselineGroup struct {
	ID         string            `yaml:"id"`
	Modules    []string          `yaml:"modules"`
	Migrations []string          `yaml:"migrations"`
	Checksums  map[string]string `yaml:"checksums"`
}

type baselineManifest struct {
	Version int             `yaml:"version"`
	Groups  []baselineGroup `yaml:"groups"`
}

type baseline struct {
	fs       fs.FS
	manifest baselineManifest
}

// loadBaseline returns nil when the bundle carries no baseline (a plain
// directory, a test FS, or a chain-only view).
func loadBaseline(migrationFS fs.FS) (*baseline, error) {
	provider, ok := migrationFS.(BaselineProvider)
	if !ok {
		return nil, nil
	}
	root := provider.BaselineFS()
	if root == nil {
		return nil, nil
	}
	data, err := fs.ReadFile(root, baselineManifestPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read migration baseline manifest: %w", err)
	}
	var manifest baselineManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse migration baseline manifest: %w", err)
	}
	if manifest.Version != baselineVersion {
		return nil, fmt.Errorf("migration baseline manifest version %d is not supported", manifest.Version)
	}
	return &baseline{fs: root, manifest: manifest}, nil
}

// script returns the group's SQL for a dialect after verifying its checksum, or
// ok=false when the baseline has no script for the dialect.
func (b *baseline) script(group baselineGroup, dialect Dialect) (script string, ok bool, err error) {
	want := group.Checksums[string(dialect)]
	if want == "" {
		return "", false, nil
	}
	content, err := fs.ReadFile(b.fs, fmt.Sprintf("%s/%s.sql", group.ID, dialect))
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read baseline %s/%s.sql: %w", group.ID, dialect, err)
	}
	sum := sha256.Sum256(content)
	if got := hex.EncodeToString(sum[:]); got != want {
		return "", false, fmt.Errorf("baseline %s/%s.sql checksum mismatch", group.ID, dialect)
	}
	return string(content), true, nil
}

// applyBaselineGroups runs the baseline script of every selected group whose
// migrations are all pending, records those migrations in the ledger and returns
// the migrations still left to run from the chain. checksums holds the ledger
// checksum of every pending migration.
func applyBaselineGroups(ctx context.Context, conn *sql.Conn, dialect Dialect, migrationFS fs.FS, enabled map[string]bool, pending []string, checksums map[string]string, applied map[string]string) ([]string, error) {
	base, err := loadBaseline(migrationFS)
	if err != nil {
		return nil, err
	}
	if base == nil || len(pending) == 0 || !base.ledgerIsGroupConsistent(applied) {
		return pending, nil
	}
	left := make(map[string]struct{}, len(pending))
	for _, id := range pending {
		left[id] = struct{}{}
	}
	for _, group := range base.manifest.Groups {
		if !baselineGroupSelected(group, enabled) || !allPending(group.Migrations, left) {
			continue
		}
		script, ok, err := base.script(group, dialect)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		apply := func(executor Executor) error {
			if _, err := executor.ExecContext(ctx, script); err != nil {
				return fmt.Errorf("failed applying baseline %s: %w", group.ID, err)
			}
			for _, id := range group.Migrations {
				if _, err := executor.ExecContext(ctx,
					"INSERT INTO schema_migrations (id, checksum) VALUES ("+placeholder(dialect, 1)+", "+placeholder(dialect, 2)+")",
					id, checksums[id],
				); err != nil {
					return fmt.Errorf("failed recording baseline migration %s: %w", id, err)
				}
			}
			return nil
		}
		if dialect == MySQL {
			// MySQL commits DDL implicitly, so there is no transaction to
			// wrap; the script is written to be safe to run again.
			err = apply(conn)
		} else {
			err = withinMigrationTransaction(ctx, conn, apply)
		}
		if err != nil {
			return nil, err
		}
		for _, id := range group.Migrations {
			delete(left, id)
		}
	}
	remaining := make([]string, 0, len(left))
	for _, id := range pending {
		if _, ok := left[id]; ok {
			remaining = append(remaining, id)
		}
	}
	return remaining, nil
}

// ledgerIsGroupConsistent reports whether no baseline group is partly applied.
func (b *baseline) ledgerIsGroupConsistent(applied map[string]string) bool {
	for _, group := range b.manifest.Groups {
		count := 0
		for _, id := range group.Migrations {
			if _, ok := applied[id]; ok {
				count++
			}
		}
		if count != 0 && count != len(group.Migrations) {
			return false
		}
	}
	return true
}

// baselineGroupSelected reports whether every module the group needs is enabled.
// A nil enabled set means every module.
func baselineGroupSelected(group baselineGroup, enabled map[string]bool) bool {
	if enabled == nil {
		return true
	}
	for _, module := range group.Modules {
		if !enabled[module] {
			return false
		}
	}
	return true
}

func allPending(ids []string, pending map[string]struct{}) bool {
	if len(ids) == 0 {
		return false
	}
	for _, id := range ids {
		if _, ok := pending[id]; !ok {
			return false
		}
	}
	return true
}

// BaselineGroupIDs lists the baseline groups of a bundle in apply order; empty
// for a bundle without a baseline. It exists for tools and tests.
func BaselineGroupIDs(migrationFS fs.FS) ([]string, error) {
	base, err := loadBaseline(migrationFS)
	if err != nil || base == nil {
		return nil, err
	}
	ids := make([]string, 0, len(base.manifest.Groups))
	for _, group := range base.manifest.Groups {
		ids = append(ids, group.ID)
	}
	sort.Strings(ids)
	return ids, nil
}

// MigrationNeeds maps every migration ID to the sorted modules that must be
// enabled for it to run: its owning module (unless core) plus the modules it
// requires. An empty list means it always runs. It is the partition key of the
// baseline groups and mirrors migrationSelected.
func MigrationNeeds(migrationFS fs.FS) (map[string][]string, error) {
	ids, err := discoverMigrations(migrationFS)
	if err != nil {
		return nil, err
	}
	needs := make(map[string][]string, len(ids))
	for _, id := range ids {
		metadata, err := readMigrationMetadata(migrationFS, id)
		if err != nil {
			return nil, err
		}
		set := map[string]struct{}{}
		if metadata.Module != "core" {
			set[metadata.Module] = struct{}{}
		}
		for _, required := range metadata.RequiresModules {
			set[required] = struct{}{}
		}
		list := make([]string, 0, len(set))
		for module := range set {
			list = append(list, module)
		}
		sort.Strings(list)
		needs[id] = list
	}
	return needs, nil
}

// ChainOnly hides a bundle's baseline so the migration chain alone builds a
// database. Schema comparison and tests use it to keep the chain as the
// independent reference the baseline is checked against.
func ChainOnly(migrationFS fs.FS) fs.FS { return chainOnlyFS{migrationFS} }

type chainOnlyFS struct{ fs.FS }
