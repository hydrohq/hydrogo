package erasure

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLogAppendIsDurableIdempotentAndDetectsTampering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "erasure.jsonl")
	log, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	intent := Intent{ID: "op-1", EditionID: "site-a", TargetID: "page-1", Kind: "erase.page"}
	first, err := log.Append(intent)
	if err != nil {
		t.Fatal(err)
	}
	second, err := log.Append(intent)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first.Sequence != 1 {
		t.Fatalf("duplicate append changed checkpoint: %#v %#v", first, second)
	}
	if _, err := log.Append(Intent{ID: "op-1", EditionID: "site-a", TargetID: "other", Kind: "erase.page"}); err == nil {
		t.Fatal("reused operation id was accepted")
	}
	if _, err := log.Append(Intent{ID: "op-2", EditionID: "site-a", TargetID: "media-1", Kind: "erase.media"}); err != nil {
		t.Fatal(err)
	}
	checkpoint, err := log.Verify()
	if err != nil || checkpoint.Sequence != 2 {
		t.Fatalf("verify = %#v/%v", checkpoint, err)
	}
	entries, latest, err := log.EntriesAfter(1)
	if err != nil || len(entries) != 1 || entries[0].ID != "op-2" || latest != checkpoint {
		t.Fatalf("entries after = %#v/%#v/%v", entries, latest, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-4] = 'x'
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := log.Verify(); !errors.Is(err, ErrCorruptLog) {
		t.Fatalf("tampered log error = %v", err)
	}
}
