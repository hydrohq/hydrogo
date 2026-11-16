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

	"gopkg.in/yaml.v3"

	"github.com/hydrohq/hydrogo/internal/collections"
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

func TestTextRelationWidgetSingletonAndActionsAreBackwardCompatible(t *testing.T) {
	initialBlog := `id: blog_posts
label: Blog
title_field: title
public: true
fields:
  - {id: title, label: Title, kind: text, required: true}
  - {id: related_products, label: Related Products, kind: text}
`
	service, types := newServiceFor(t, initialBlog)
	created, err := service.CreateRecord(context.Background(), "blog_posts", map[string]any{
		"title":            "Leather Guide",
		"related_products": "classic-belt, brass-buckle",
	})
	if err != nil {
		t.Fatal(err)
	}

	upgradedBlog := `id: blog_posts
label: Blog
title_field: title
public: true
actions:
  - {label: "Open on site", href: "/blog/{slug}"}
fields:
  - {id: title, label: Title, kind: text, required: true}
  - {id: related_products, label: Related Products, kind: text, widget: relation, to: products, many: true, value_field: slug}
`
	singletonCompany := `id: company_info
label: Company Info
singleton: true
public: true
fields:
  - {id: unvan, label: Name, kind: text}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "blog_posts.yaml"), []byte(upgradedBlog), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "company_info.yaml"), []byte(singletonCompany), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	schemas, err := Schemas(loaded)
	if err != nil {
		t.Fatalf("Schemas: %v", err)
	}
	if err := service.SyncSchemas(context.Background(), schemas); err != nil {
		t.Fatalf("SyncSchemas must keep existing text records intact: %v", err)
	}
	got, err := service.GetRecord(context.Background(), "blog_posts", created.ID)
	if err != nil || got.Data["related_products"] != "classic-belt, brass-buckle" {
		t.Fatalf("got=%+v err=%v", got, err)
	}

	collectionsMeta := AdminCollections(loaded, false)
	var blogCol, compCol int = -1, -1
	for i, c := range collectionsMeta {
		if c.ID == "blog_posts" {
			blogCol = i
		}
		if c.ID == "company_info" {
			compCol = i
		}
	}
	if blogCol < 0 || compCol < 0 {
		t.Fatalf("missing admin collections: %+v", collectionsMeta)
	}
	if collectionsMeta[compCol].Type != "global" {
		t.Fatalf("company_info Type = %q, want global", collectionsMeta[compCol].Type)
	}
	if len(collectionsMeta[blogCol].Actions) != 1 || collectionsMeta[blogCol].Actions[0].Href != "/blog/{slug}" {
		t.Fatalf("blog_posts Actions = %+v", collectionsMeta[blogCol].Actions)
	}
	relField := collectionsMeta[blogCol].Fields[1]
	if relField.Kind != "text" || relField.Widget != "relation" || relField.Relation == nil || relField.Relation.Collection != "products" || relField.Relation.Cardinality != "many" || relField.Relation.ValueField != "slug" {
		t.Fatalf("relField = %+v", relField)
	}
	_ = types

	// External action hrefs and non-relation value_field must fail validation.
	for name, bad := range map[string]string{
		"external action": "id: x\nlabel: X\nactions: [{label: Bad, href: https://evil.test}]\nfields: [{id: a, label: A, kind: text}]\n",
		"bad value_field": "id: x\nlabel: X\nfields: [{id: a, label: A, kind: text, value_field: slug}]\n",
	} {
		t.Run(name, func(t *testing.T) {
			badDir := t.TempDir()
			_ = os.WriteFile(filepath.Join(badDir, "x.yaml"), []byte(bad), 0o644)
			if _, err := LoadDir(badDir); err == nil {
				t.Fatalf("%s unexpectedly accepted", name)
			}
		})
	}
}

func TestDefinitionYAMLMarshallingOmitsZeroFields(t *testing.T) {
	def := Definition{
		ID:     "notes",
		Label:  "Notes",
		Public: false,
		Fields: []Field{
			{ID: "title", Label: "Title", Kind: "text", Required: true},
		},
	}
	raw, err := yaml.Marshal(&def)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	text := string(raw)
	for _, unwanted := range []string{"drafts:", "revisions:", "permissions:", "singleton:", "searchable:", "filterable:", "localized:", "null"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("unexpected zero field %q in marshalled YAML:\n%s", unwanted, text)
		}
	}
	roundDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(roundDir, "notes.yaml"), raw, 0o644); err != nil {
		t.Fatalf("write roundtrip yaml: %v", err)
	}
	loaded, err := LoadDir(roundDir)
	if err != nil || len(loaded) != 1 || loaded[0].ID != "notes" {
		t.Fatalf("LoadDir roundtrip failed: loaded=%+v err=%v", loaded, err)
	}
}
