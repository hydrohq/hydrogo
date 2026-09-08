package erasure

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplayRetriesFromCheckpointAfterFailure(t *testing.T) {
	log, err := New(filepath.Join(t.TempDir(), "erasure.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"one", "two"} {
		if _, err := log.Append(Intent{ID: id, EditionID: "edition", TargetID: id, Kind: "page"}); err != nil {
			t.Fatal(err)
		}
	}
	var seen []string
	fail := errors.New("temporary deletion failure")
	if _, err := log.Replay(context.Background(), 0, func(_ context.Context, intent Intent) error {
		seen = append(seen, intent.ID)
		if intent.ID == "two" {
			return fail
		}
		return nil
	}); !errors.Is(err, fail) {
		t.Fatalf("replay error = %v, want %v", err, fail)
	}
	if got := strings.Join(seen, ","); got != "one,two" {
		t.Fatalf("first replay order = %q", got)
	}
	seen = nil
	checkpoint, err := log.Replay(context.Background(), 1, func(_ context.Context, intent Intent) error {
		seen = append(seen, intent.ID)
		return nil
	})
	if err != nil || checkpoint.Sequence != 2 {
		t.Fatalf("retry checkpoint = %+v/%v", checkpoint, err)
	}
	if got := strings.Join(seen, ","); got != "two" {
		t.Fatalf("retry order = %q", got)
	}
}

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
