package contenttypes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	"github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

const faqYAML = `id: faq
label: FAQ
title_field: question
public: true
fields:
  - {id: question, label: Question, kind: text, required: true, list: true, unique: true}
  - {id: answer, label: Answer, kind: richtext}
  - {id: asked_on, label: Asked on, kind: date}
  - {id: sort_order, label: Order, kind: number}
`

func TestLoadDirValidatesDefinitions(t *testing.T) {
	cases := map[string]string{
		"unknown key":        "id: x\nlabel: X\ncolour: red\nfields: [{id: a, label: A, kind: text}]\n",
		"bad id":             "id: X-Y\nlabel: X\nfields: [{id: a, label: A, kind: text}]\n",
		"reserved field":     "id: x\nlabel: X\nfields: [{id: version, label: V, kind: text}]\n",
		"unknown kind":       "id: x\nlabel: X\nfields: [{id: a, label: A, kind: colour}]\n",
		"relation w/o to":    "id: x\nlabel: X\nfields: [{id: a, label: A, kind: relation}]\n",
		"select w/o options": "id: x\nlabel: X\nfields: [{id: a, label: A, kind: select}]\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "x.yaml"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := LoadDir(dir); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "faq.yaml"), []byte(faqYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	types, err := LoadDir(dir)
	if err != nil || len(types) != 1 || types[0].ID != "faq" {
		t.Fatalf("types=%+v err=%v", types, err)
	}
	if missing, err := LoadDir(filepath.Join(dir, "missing")); err != nil || len(missing) != 0 {
		t.Fatalf("missing dir: types=%v err=%v", missing, err)
	}
}

func TestShippedContentTypesAreValid(t *testing.T) {
	types, err := LoadDir(filepath.Join("..", "..", "content-types"))
	if err != nil {
		t.Fatal(err)
	}
	if len(types) == 0 {
		t.Fatal("no shipped content types found")
	}
}

func newService(t *testing.T) (collections.Service, []Definition) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "content.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "faq.yaml"), []byte(faqYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	types, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := Schemas(types)
	if err != nil {
		t.Fatal(err)
	}
	service := collections.Service{Repository: sqlstore.Store{DB: db, Dialect: database.SQLite}}
	if err := service.SyncSchemas(ctx, schemas); err != nil {
		t.Fatal(err)
	}
	return service, types
}

func call(t *testing.T, h http.Handler, method, path, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, request)
	var decoded map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &decoded)
	return recorder, decoded
}

func TestAdminAPIServesTheGeneratedAdminContract(t *testing.T) {
	service, types := newService(t)
	api := AdminAPI{Service: service, Types: types}

	created, record := call(t, api, http.MethodPost, "/api/v1/faq", `{"id":"ignored","question":"How?","asked_on":"2026-10-14","sort_order":null}`)
	if created.Code != http.StatusCreated || record["question"] != "How?" || record["version"] != float64(1) {
		t.Fatalf("create: %d %s", created.Code, created.Body)
	}
	if record["asked_on"] != "2026-10-14" {
		t.Fatalf("date was not returned in date-input format: %v", record["asked_on"])
	}
	id := record["id"].(string)
	if id == "ignored" {
		t.Fatal("client-supplied id was used")
	}

	list := httptest.NewRecorder()
	api.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/faq?limit=26&offset=0", nil))
	var items []map[string]any
	if err := json.Unmarshal(list.Body.Bytes(), &items); err != nil || len(items) != 1 || items[0]["id"] != id {
		t.Fatalf("list: %d %s", list.Code, list.Body)
	}

	updated, record := call(t, api, http.MethodPut, "/api/v1/faq/"+id, `{"id":"`+id+`","version":1,"question":"How exactly?"}`)
	if updated.Code != http.StatusOK || record["version"] != float64(2) {
		t.Fatalf("update: %d %s", updated.Code, updated.Body)
	}
	if stale, _ := call(t, api, http.MethodPut, "/api/v1/faq/"+id, `{"version":1,"question":"Stale"}`); stale.Code != http.StatusConflict {
		t.Fatalf("stale update: %d %s", stale.Code, stale.Body)
	}

	invalid, payload := call(t, api, http.MethodPost, "/api/v1/faq", `{"answer":"no question"}`)
	if invalid.Code != http.StatusUnprocessableEntity || payload["code"] != "validation_failed" {
		t.Fatalf("missing required: %d %s", invalid.Code, invalid.Body)
	}
	if fields := payload["fields"].(map[string]any); fields["question"] != "required" {
		t.Fatalf("fields=%v", fields)
	}
	duplicate, payload := call(t, api, http.MethodPost, "/api/v1/faq", `{"question":"How exactly?"}`)
	if duplicate.Code != http.StatusUnprocessableEntity || payload["fields"].(map[string]any)["question"] != "taken" {
		t.Fatalf("duplicate: %d %s", duplicate.Code, duplicate.Body)
	}

	if missing, _ := call(t, api, http.MethodGet, "/api/v1/nope", ""); missing.Code != http.StatusNotFound {
		t.Fatalf("unknown type: %d", missing.Code)
	}
	if deleted, _ := call(t, api, http.MethodDelete, "/api/v1/faq/"+id, ""); deleted.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", deleted.Code, deleted.Body)
	}
}

func TestPublicAPIOnlyExposesPublicTypes(t *testing.T) {
	service, types := newService(t)
	if _, err := service.CreateRecord(context.Background(), "faq", map[string]any{"question": "Public?"}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	public := PublicAPI{Service: service, Types: types}
	mux.HandleFunc("GET /api/v1/content/{type}", public.List)
	mux.HandleFunc("GET /api/v1/content/{type}/{id}", public.Get)

	ok, body := call(t, mux, http.MethodGet, "/api/v1/content/faq", "")
	if ok.Code != http.StatusOK || len(body["items"].([]any)) != 1 {
		t.Fatalf("public list: %d %s", ok.Code, ok.Body)
	}
	types[0].Public = false
	if hidden, _ := call(t, mux, http.MethodGet, "/api/v1/content/faq", ""); hidden.Code != http.StatusNotFound {
		t.Fatalf("private type: %d", hidden.Code)
	}
}
