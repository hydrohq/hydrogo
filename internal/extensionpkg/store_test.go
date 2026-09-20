package extensionpkg

import (
	"path/filepath"
	"testing"

	extensionv1 "github.com/ardaozyildirim/hydrogo/extension/v1"
)

func TestStorePersistsAtomicLifecycleState(t *testing.T) {
	store := Store{Root: t.TempDir()}
	record := Record{ID: "acme-sales", Version: "1.0.0", State: extensionv1.StateInstalled, Artifact: filepath.Join("acme-sales", "1.0.0.hx")}
	if err := store.Save(record); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Transition(loaded, extensionv1.StateApproved); err != nil {
		t.Fatal(err)
	}
	if loaded, err = store.Load(record.ID); err != nil || loaded.State != extensionv1.StateApproved {
		t.Fatalf("loaded = %#v, err = %v", loaded, err)
	}
}
