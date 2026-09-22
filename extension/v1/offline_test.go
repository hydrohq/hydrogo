package v1

import "testing"

func TestValidateOfflineSyncBatchIsBoundedAndDeterministic(t *testing.T) {
	batch, err := ValidateOfflineSyncBatch(OfflineSyncBatch{Cursor: "site:1", Commands: []OfflineCommand{
		{ID: "b", Type: "inspection.update", Payload: []byte("{}"), IdempotencyKey: "key-b"},
		{ID: "a", Type: "inspection.create", Payload: []byte("{}"), IdempotencyKey: "key-a"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if batch.Commands[0].ID != "a" || batch.Commands[1].ID != "b" {
		t.Fatalf("commands not ordered: %#v", batch.Commands)
	}
	if _, err := ValidateOfflineSyncBatch(OfflineSyncBatch{Commands: []OfflineCommand{{ID: "a", Type: "x", IdempotencyKey: "k"}, {ID: "a", Type: "x", IdempotencyKey: "k2"}}}); err == nil {
		t.Fatal("duplicate command accepted")
	}
}
