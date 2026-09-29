package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	storagesqlite "github.com/ardaozyildirim/hydrogo/internal/storage/sqlite"
	"gopkg.in/yaml.v3"
)

type metadata struct {
	ID            string            `yaml:"id"`
	Module        string            `yaml:"module"`
	Dependencies  []string          `yaml:"dependencies"`
	Checksums     map[string]string `yaml:"checksum_sha256"`
	Transactional map[string]bool   `yaml:"transactional"`
	LockTimeout   string            `yaml:"lock_timeout"`
	DiskEstimate  string            `yaml:"disk_estimate"`
	Compatibility string            `yaml:"compatibility"`
	Rollback      string            `yaml:"rollback"`
	SQLiteShadow  *shadowMetadata   `yaml:"sqlite_shadow"`
}

type shadowMetadata struct {
	Table     string                       `yaml:"table"`
	CreateSQL string                       `yaml:"create_sql"`
	Columns   []storagesqlite.ShadowColumn `yaml:"columns"`
	BatchSize int                          `yaml:"batch_size"`
	AfterSwap []string                     `yaml:"after_swap"`
	Checksum  string                       `yaml:"checksum_sha256"`
}

func (m shadowMetadata) spec() storagesqlite.ShadowMigration {
	return storagesqlite.ShadowMigration{Table: m.Table, CreateSQL: m.CreateSQL, Columns: m.Columns, BatchSize: m.BatchSize, AfterSwap: m.AfterSwap}
}

func main() {
	root := flag.String("root", "migrations", "migration root")
	flag.Parse()
	entries, err := os.ReadDir(*root)
	if err != nil {
		fail(err)
	}
	known := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() {
			known[entry.Name()] = true
		}
	}
	var failures []string
	for id := range known {
		data, err := os.ReadFile(filepath.Join(*root, id, "migration.yaml"))
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: metadata: %v", id, err))
			continue
		}
		var m metadata
		if err := yaml.Unmarshal(data, &m); err != nil {
			failures = append(failures, fmt.Sprintf("%s: invalid yaml: %v", id, err))
			continue
		}
		if m.ID != id {
			failures = append(failures, fmt.Sprintf("%s: id=%q", id, m.ID))
		}
		if strings.TrimSpace(m.Module) == "" {
			failures = append(failures, fmt.Sprintf("%s: module is required", id))
		}
		if m.Checksums["sqlite"] == "" || m.Checksums["postgres"] == "" {
			failures = append(failures, fmt.Sprintf("%s: sqlite/postgres checksums are required", id))
		}
		if m.SQLiteShadow != nil {
			shadowChecksum, checksumErr := storagesqlite.Checksum(m.SQLiteShadow.spec())
			if checksumErr != nil || !strings.EqualFold(shadowChecksum, m.SQLiteShadow.Checksum) {
				failures = append(failures, fmt.Sprintf("%s: SQLite shadow descriptor checksum mismatch", id))
			}
			if transactional, ok := m.Transactional["sqlite"]; !ok || transactional {
				failures = append(failures, fmt.Sprintf("%s: SQLite shadow migration must be nontransactional", id))
			}
		}
		if _, ok := m.Transactional["sqlite"]; !ok {
			failures = append(failures, fmt.Sprintf("%s: transactional.sqlite is required", id))
		}
		if _, ok := m.Transactional["postgres"]; !ok {
			failures = append(failures, fmt.Sprintf("%s: transactional.postgres is required", id))
		}
		if _, err := os.Stat(filepath.Join(*root, id, "mysql.sql")); err == nil {
			if m.Checksums["mysql"] == "" {
				failures = append(failures, fmt.Sprintf("%s: mysql checksum is required", id))
			}
			if _, ok := m.Transactional["mysql"]; !ok {
				failures = append(failures, fmt.Sprintf("%s: transactional.mysql is required", id))
			}
		} else if !os.IsNotExist(err) {
			failures = append(failures, fmt.Sprintf("%s: mysql SQL: %v", id, err))
		}
		for field, value := range map[string]string{"lock_timeout": m.LockTimeout, "disk_estimate": m.DiskEstimate, "compatibility": m.Compatibility, "rollback": m.Rollback} {
			if strings.TrimSpace(value) == "" {
				failures = append(failures, fmt.Sprintf("%s: %s is required", id, field))
			}
		}
		for _, dep := range m.Dependencies {
			if !known[dep] {
				failures = append(failures, fmt.Sprintf("%s: unknown dependency %q", id, dep))
			}
		}
	}
	sort.Strings(failures)
	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Fprintln(os.Stderr, "migrationguard:", failure)
		}
		os.Exit(1)
	}
	fmt.Printf("migrationguard: OK migrations=%d\n", len(known))
}

func fail(err error) { fmt.Fprintln(os.Stderr, "migrationguard:", err); os.Exit(1) }
