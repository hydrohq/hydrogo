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
	return newServiceFor(t, faqYAML)
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
	if ok.Code != http.StatusOK || len(body["data"].([]any)) != 1 {
		t.Fatalf("public list: %d %s", ok.Code, ok.Body)
	}
	types[0].Public = false
	if hidden, _ := call(t, mux, http.MethodGet, "/api/v1/content/faq", ""); hidden.Code != http.StatusNotFound {
		t.Fatalf("private type: %d", hidden.Code)
	}
}
