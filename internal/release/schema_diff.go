package release

import (
	"fmt"
	"sort"
)

// MigrationDiff describes changes between two release migration ledgers.
// Additions are compatible; removals and checksum changes are breaking.
type MigrationDiff struct {
	Added   []string
	Removed []string
	Changed []string
}

// CompareMigrationChecksums compares release-level migration checksums without
// exposing SQL dialect details. Existing migration IDs must remain immutable.
func CompareMigrationChecksums(baseline, current map[string]string) (MigrationDiff, error) {
	var diff MigrationDiff
	for id, checksum := range baseline {
		currentChecksum, ok := current[id]
		if !ok {
			diff.Removed = append(diff.Removed, id)
			continue
		}
		if checksum == "" || currentChecksum == "" || checksum != currentChecksum {
			diff.Changed = append(diff.Changed, id)
		}
	}
	for id := range current {
		if _, ok := baseline[id]; !ok {
			diff.Added = append(diff.Added, id)
		}
	}
	sort.Strings(diff.Added)
	sort.Strings(diff.Removed)
	sort.Strings(diff.Changed)
	if len(diff.Removed) != 0 || len(diff.Changed) != 0 {
		return diff, fmt.Errorf("migration ledger is incompatible: removed=%d changed=%d", len(diff.Removed), len(diff.Changed))
	}
	return diff, nil
}
