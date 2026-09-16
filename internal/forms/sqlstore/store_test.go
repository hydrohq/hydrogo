package sqlstore

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/forms"
)

func newFormStore(t *testing.T) Store {
	t.Helper()
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "forms.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE hydro_forms (id TEXT PRIMARY KEY,title TEXT NOT NULL,schema_json TEXT NOT NULL,updated_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	return Store{Executor: db, Dialect: database.SQLite}
}

func TestFormStoreRoundTripListAndDelete(t *testing.T) {
	s := newFormStore(t)
	form := forms.Form{ID: "contact", Title: "Contact", Fields: []forms.Field{{ID: "email", Type: forms.Email, Label: "Email", Required: true}}}
	if err := s.Upsert(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(context.Background(), "contact")
	if err != nil || got.ID != form.ID || got.Fields[0].Type != forms.Email {
		t.Fatalf("get=%#v err=%v", got, err)
	}
	list, err := s.List(context.Background())
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%#v err=%v", list, err)
	}
	if err := s.Delete(context.Background(), "contact"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(context.Background(), "contact"); !errors.Is(err, ErrFormNotFound) {
		t.Fatalf("missing delete=%v", err)
	}
}

func TestFormStoreRejectsInvalidForm(t *testing.T) {
	s := newFormStore(t)
	if err := s.Upsert(context.Background(), forms.Form{ID: "bad id", Title: "x"}); err == nil {
		t.Fatal("invalid form accepted")
	}
}
