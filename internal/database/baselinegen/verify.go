package baselinegen

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type manifestFile struct {
	Version int `yaml:"version"`
	Groups  []struct {
		ID         string            `yaml:"id"`
		Modules    []string          `yaml:"modules"`
		Migrations []string          `yaml:"migrations"`
		Checksums  map[string]string `yaml:"checksums"`
	} `yaml:"groups"`
}

// VerifyManifest checks the baseline against the migration chain without a
// database and returns every problem it finds: the manifest parses, each group's
// migrations exist, belong to that group and to no other, the scripts exist and
// match their checksums, and SQLite and PostgreSQL scripts are present. A
// migration the baseline does not list is fine: it is newer than the baseline
// and runs from the chain.
func VerifyManifest(migrationFS, baselineFS fs.FS) []string {
	data, err := fs.ReadFile(baselineFS, "manifest.yml")
	if err != nil {
		return []string{"manifest.yml: " + err.Error()}
	}
	var manifest manifestFile
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return []string{"manifest.yml: " + err.Error()}
	}
	var problems []string
	if manifest.Version != 1 {
		problems = append(problems, fmt.Sprintf("manifest version %d is not supported", manifest.Version))
	}
	groups, err := Groups(migrationFS)
	if err != nil {
		return append(problems, err.Error())
	}
	groupOf := map[string]string{}
	for _, group := range groups {
		for _, migration := range group.Migrations {
			groupOf[migration] = group.ID
		}
	}
	seenGroup := map[string]bool{}
	covered := map[string]string{}
	for _, group := range manifest.Groups {
		if seenGroup[group.ID] {
			problems = append(problems, "group "+group.ID+" is listed twice")
		}
		seenGroup[group.ID] = true
		if len(group.Migrations) == 0 {
			problems = append(problems, "group "+group.ID+" has no migrations")
		}
		if !sort.StringsAreSorted(group.Migrations) {
			problems = append(problems, "group "+group.ID+": migrations are not sorted")
		}
		if want := GroupID(group.Modules); want != group.ID {
			problems = append(problems, fmt.Sprintf("group %s: modules %v name the group %s", group.ID, group.Modules, want))
		}
		for _, migration := range group.Migrations {
			if other, dup := covered[migration]; dup {
				problems = append(problems, fmt.Sprintf("migration %s is in groups %s and %s", migration, other, group.ID))
			}
			covered[migration] = group.ID
			actual, exists := groupOf[migration]
			switch {
			case !exists:
				problems = append(problems, fmt.Sprintf("group %s lists %s, which is not a migration", group.ID, migration))
			case actual != group.ID:
				problems = append(problems, fmt.Sprintf("migration %s belongs to group %s, not %s", migration, actual, group.ID))
			}
		}
		for _, dialect := range []string{"sqlite", "postgres"} {
			if group.Checksums[dialect] == "" {
				problems = append(problems, fmt.Sprintf("group %s has no %s script", group.ID, dialect))
			}
		}
		for dialect, want := range group.Checksums {
			content, err := fs.ReadFile(baselineFS, group.ID+"/"+dialect+".sql")
			if errors.Is(err, fs.ErrNotExist) {
				problems = append(problems, fmt.Sprintf("group %s: %s.sql is missing", group.ID, dialect))
				continue
			}
			if err != nil {
				problems = append(problems, err.Error())
				continue
			}
			digest := sha256.Sum256(content)
			if hex.EncodeToString(digest[:]) != want {
				problems = append(problems, fmt.Sprintf("group %s: %s.sql does not match its checksum", group.ID, dialect))
			}
		}
	}
	// Prerequisites come first: core, then smaller module sets.
	for index := 1; index < len(manifest.Groups); index++ {
		previous, current := manifest.Groups[index-1], manifest.Groups[index]
		if len(previous.Modules) > len(current.Modules) {
			problems = append(problems, fmt.Sprintf("group %s must come before %s", current.ID, previous.ID))
		}
	}
	if len(manifest.Groups) > 0 && manifest.Groups[0].ID != "core" {
		problems = append(problems, "the first group must be core")
	}
	sort.Strings(problems)
	return problems
}

// Describe summarizes a problem list for a failure message.
func Describe(problems []string) string { return strings.Join(problems, "\n") }
