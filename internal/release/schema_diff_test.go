package release

import "testing"

func TestCompareMigrationChecksumsAllowsAdditions(t *testing.T) {
	diff, err := CompareMigrationChecksums(map[string]string{"001": "a"}, map[string]string{"001": "a", "002": "b"})
	if err != nil || len(diff.Added) != 1 || diff.Added[0] != "002" {
		t.Fatalf("diff=%#v err=%v", diff, err)
	}
}

func TestCompareMigrationChecksumsRejectsRemovalAndChange(t *testing.T) {
	if _, err := CompareMigrationChecksums(map[string]string{"001": "a", "002": "b"}, map[string]string{"001": "changed"}); err == nil {
		t.Fatal("expected incompatible migration ledger")
	}
}
