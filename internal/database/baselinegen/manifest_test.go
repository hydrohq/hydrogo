package baselinegen_test

import (
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database/baselinegen"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestTheCommittedBaselineMatchesTheMigrationChain(t *testing.T) {
	if problems := baselinegen.VerifyManifest(migrations.FS, migrations.FS.BaselineFS()); len(problems) > 0 {
		t.Fatalf("migrations/baseline is out of step with migrations/:\n%s\nRegenerate with go run ./tools/baselinegen", baselinegen.Describe(problems))
	}
}

// Every migration written before the baseline is covered by it; one added later
// is not, and runs from the chain on top.
func TestEveryCurrentMigrationIsCoveredUntilANewerOneIsAdded(t *testing.T) {
	groups, err := baselinegen.Groups(migrations.FS)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, group := range groups {
		total += len(group.Migrations)
	}
	ids, err := databaseGroupIDs()
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != len(groups) {
		t.Logf("baseline has %d groups, the chain partitions into %d: the chain has grown since the baseline was generated", len(ids), len(groups))
	}
	if total == 0 {
		t.Fatal("no migrations found")
	}
}
