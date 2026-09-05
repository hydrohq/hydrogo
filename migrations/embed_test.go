package migrations

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestExperimentMetadataChecksumsMatchSQL(t *testing.T) {
	metadata, err := FS.ReadFile("202609060001_transaction_contract/migration.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, dialect := range []string{"sqlite", "postgres"} {
		content, err := FS.ReadFile("202609060001_transaction_contract/" + dialect + ".sql")
		if err != nil {
			t.Fatal(err)
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(content))
		want := dialect + ": \"" + checksum + "\""
		if !strings.Contains(string(metadata), want) {
			t.Fatalf("metadata does not contain %s checksum %s", dialect, checksum)
		}
	}
}
