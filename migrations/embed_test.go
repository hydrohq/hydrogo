package migrations

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"testing"
)

func TestExperimentMetadataChecksumsMatchSQL(t *testing.T) {
	entries, err := fs.ReadDir(FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		metadata, err := FS.ReadFile(id + "/migration.yaml")
		if err != nil {
			t.Fatalf("%s metadata: %v", id, err)
		}
		for _, dialect := range []string{"sqlite", "postgres"} {
			content, err := FS.ReadFile(id + "/" + dialect + ".sql")
			if err != nil {
				t.Fatalf("%s %s SQL: %v", id, dialect, err)
			}
			checksum := fmt.Sprintf("%x", sha256.Sum256(content))
			want := dialect + ": \"" + checksum + "\""
			if !strings.Contains(string(metadata), want) {
				t.Fatalf("%s metadata does not contain %s checksum %s", id, dialect, checksum)
			}
		}
	}
}
