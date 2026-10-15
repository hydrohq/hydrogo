package contenttypes

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

const draftsFAQ = `id: faq
label: FAQ
title_field: question
public: true
drafts: true
fields:
  - {id: question, label: Question, kind: text, required: true, list: true, searchable: true}
  - {id: category, label: Category, kind: select, options: [general, billing], list: true, filterable: true}
`

func publicMux(service PublicAPI) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/content/{type}", service.List)
	mux.HandleFunc("GET /api/v1/content/{type}/{id}", service.Get)
	return mux
}

func publicCount(t *testing.T, mux http.Handler, target string) int {
	t.Helper()
	recorder, body := call(t, mux, http.MethodGet, target, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s: %d %s", target, recorder.Code, recorder.Body)
	}
	return len(body["data"].([]any))
}

func TestDraftsRejectAStatusField(t *testing.T) {
	dir := t.TempDir()
	content := "id: x\nlabel: X\ndrafts: true\nfields: [{id: status, label: S, kind: text}]\n"
	if err := os.WriteFile(filepath.Join(dir, "x.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDir(dir); err == nil {
		t.Fatal("a drafts type accepted its own status field")
	}
}

func TestRecordsWithoutStatusAreDrafts(t *testing.T) {
	service, types := newServiceFor(t, draftsFAQ)
	record, err := service.CreateRecord(context.Background(), "faq", map[string]any{"question": "Old record"})
	if err != nil {
		t.Fatal(err)
	}
	public := publicMux(PublicAPI{Service: service, Types: types})
	if count := publicCount(t, public, "/api/v1/content/faq"); count != 0 {
		t.Fatalf("record without status is public: %d", count)
	}
	admin := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})
	if recorder, body := call(t, admin, http.MethodGet, AdminAPIPath+"/faq/"+record.ID, ""); recorder.Code != http.StatusOK || body["data"].(map[string]any)[StatusField] != StatusDraft {
		t.Fatalf("admin view: %d %v", recorder.Code, body)
	}
}
