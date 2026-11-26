package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hydrohq/hydrogo/internal/adminstudio"
	"github.com/hydrohq/hydrogo/internal/auth"
	"github.com/hydrohq/hydrogo/internal/config"
	"github.com/hydrohq/hydrogo/internal/contenttypes"
)

func TestAdminSchemaListsContentTypesAndModules(t *testing.T) {
	dir := t.TempDir()
	faq := "id: faq\nlabel: FAQ\ntitle_field: question\npublic: true\ndrafts: true\nrevisions: true\nfields:\n  - {id: question, label: Question, kind: text, required: true, list: true, localized: true}\n"
	if err := os.WriteFile(filepath.Join(dir, "faq.yaml"), []byte(faq), 0o644); err != nil {
		t.Fatal(err)
	}
	types, err := contenttypes.LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	registry, err := adminstudio.NewRegistry([]adminstudio.Collection{
		{ID: "pages", Label: "Pages", Source: adminstudio.SourceModule, APIPath: "/api/v1/admin/pages", Fields: []adminstudio.Field{{ID: "title", Label: "Title", Kind: adminstudio.FieldText}}},
		{ID: "news", Label: "News", Source: adminstudio.SourceModule, APIPath: "/api/v1/admin/news", Fields: []adminstudio.Field{{ID: "title", Label: "Title", Kind: adminstudio.FieldText}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	app := &serveApp{
		modules:        map[string]bool{"pages": true, "news": false, "collections": true},
		studioRegistry: registry,
		contentTypes:   types,
		contentLocales: contenttypes.NewLocales(true, "tr", []string{"tr", "en"}),
	}

	anonymous := httptest.NewRecorder()
	app.adminSchema(anonymous, httptest.NewRequest(http.MethodGet, "/api/v1/admin/schema", nil))
	if anonymous.Code != http.StatusForbidden {
		t.Fatalf("anonymous schema: %d", anonymous.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/schema", nil)
	request = request.WithContext(auth.WithUser(request.Context(), &auth.User{ID: "u1", Role: auth.RoleEditor, Active: true}))
	recorder := httptest.NewRecorder()
	app.adminSchema(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("schema: %d %s", recorder.Code, recorder.Body)
	}
	var body struct {
		Data adminSchema `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	schema := body.Data
	if schema.Version != 1 || len(schema.Resources) != 2 {
		t.Fatalf("schema=%+v", schema)
	}
	byID := map[string]adminstudio.Collection{}
	for _, resource := range schema.Resources {
		byID[resource.ID] = resource
	}
	if byID["faq"].Source != adminstudio.SourceContentType || byID["faq"].APIPath != "/api/v1/admin/content/faq" || !byID["faq"].Localization || byID["faq"].Workflow == nil {
		t.Fatalf("faq resource=%+v", byID["faq"])
	}
	if byID["pages"].Source != adminstudio.SourceModule || byID["pages"].APIPath != "/api/v1/admin/pages" {
		t.Fatalf("pages resource=%+v", byID["pages"])
	}
	if _, listed := byID["news"]; listed {
		t.Fatal("disabled module listed in the schema")
	}
	if len(schema.ContentTypes) != 1 || schema.ContentTypes[0].ID != "faq" || !schema.ContentTypes[0].Drafts || !schema.ContentTypes[0].Localization {
		t.Fatalf("content types=%+v", schema.ContentTypes)
	}
	if !schema.Modules["pages"] || schema.Modules["news"] {
		t.Fatalf("modules=%v", schema.Modules)
	}
	if !schema.Locales.Enabled || schema.Locales.Default != "tr" || len(schema.Locales.All) != 2 {
		t.Fatalf("locales=%+v", schema.Locales)
	}
	if len(schema.Permissions) == 0 {
		t.Fatal("permission catalog missing")
	}
	// The admin asks for hook services only when some are configured.
	if schema.Modules["hooks"] {
		t.Fatal("hooks listed without any configured")
	}
	app.cfg.Hooks = []config.SyncHook{{ID: "ledger"}}
	if !app.buildAdminSchema().Modules["hooks"] {
		t.Fatal("configured hook services are not announced to the admin")
	}
}

func TestAdminBrandingIsPublicAndEmptyWithoutTheSiteModule(t *testing.T) {
	app := &serveApp{modules: map[string]bool{"pages": true}}
	recorder := httptest.NewRecorder()
	app.adminBrandingHandler(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/admin/branding", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("branding: %d %s", recorder.Code, recorder.Body)
	}
	var body struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"name", "logo_url", "color", "favicon_url", "default_locale"} {
		if value, ok := body.Data[key]; !ok || value != "" {
			t.Errorf("%s = %q (present %v), want empty", key, value, ok)
		}
	}
}

func TestAdminSchemaPlatformAndSiteURL(t *testing.T) {
	app := &serveApp{modules: map[string]bool{"pages": true}}
	editor := &auth.User{ID: "u1", Role: auth.RoleEditor, Active: true}

	// 1. Defaults: platform is false, site_url is empty
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/schema", nil)
	req = req.WithContext(auth.WithUser(req.Context(), editor))
	rec := httptest.NewRecorder()
	app.adminSchema(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	var res struct {
		Data adminSchema `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.Data.Platform {
		t.Errorf("expected platform to be false by default, got true")
	}
	if res.Data.SiteURL != "" {
		t.Errorf("expected site_url to be empty by default, got %q", res.Data.SiteURL)
	}

	// 2. SiteURL from http.site_url
	app.cfg.HTTP.SiteURL = "http://localhost:3000"
	app.cfg.HTTP.PublicOrigin = "https://public.example.com"
	rec = httptest.NewRecorder()
	app.adminSchema(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Data.SiteURL != "http://localhost:3000" {
		t.Errorf("expected site_url http://localhost:3000, got %q", res.Data.SiteURL)
	}

	// 3. Fallback to public_origin when site_url is empty
	app.cfg.HTTP.SiteURL = ""
	rec = httptest.NewRecorder()
	app.adminSchema(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Data.SiteURL != "https://public.example.com" {
		t.Errorf("expected site_url fallback to public_origin, got %q", res.Data.SiteURL)
	}

	// 4. Platform mode true when Factory.BaseDomain is set
	app.cfg.Factory.BaseDomain = "boutique.dev"
	rec = httptest.NewRecorder()
	app.adminSchema(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if !res.Data.Platform {
		t.Errorf("expected platform to be true with BaseDomain, got false")
	}

	// 5. Platform mode true when Tenancy.Enabled is set
	app.cfg.Factory.BaseDomain = ""
	app.cfg.Tenancy.Enabled = true
	rec = httptest.NewRecorder()
	app.adminSchema(rec, req)
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if !res.Data.Platform {
		t.Errorf("expected platform to be true with Tenancy.Enabled, got false")
	}
}
