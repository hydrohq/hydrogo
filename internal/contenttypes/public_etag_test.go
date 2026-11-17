package contenttypes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hydrohq/hydrogo/internal/collections"
	"github.com/hydrohq/hydrogo/internal/collections/sqlstore"
	"github.com/hydrohq/hydrogo/internal/database"
	"github.com/hydrohq/hydrogo/internal/tenant"
	"github.com/hydrohq/hydrogo/migrations"
)

const publicArticlesYAML = `id: articles
label: Articles
public: true
fields:
  - {id: title, label: Title, kind: text, required: true, list: true}
`

func TestPublicAPIETagAndNotModified(t *testing.T) {
	service, types := newServiceFor(t, publicArticlesYAML)
	adminHandler := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})

	// Create article
	r, res := call(t, adminHandler, http.MethodPost, AdminAPIPath+"/articles", `{"title":"Performance First"}`)
	if r.Code != http.StatusCreated {
		t.Fatalf("seed: %d", r.Code)
	}
	dataMap, ok := res["data"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected res format: %v", res)
	}
	id, _ := dataMap["id"].(string)
	if id == "" {
		t.Fatal("missing record id")
	}

	pubHandler := baasPublicMux(PublicAPI{Service: service, Types: types})

	// 1. Initial GET /api/v1/content/articles/{id}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/content/articles/"+id, nil)
	recW := httptest.NewRecorder()
	pubHandler.ServeHTTP(recW, req)

	if recW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", recW.Code)
	}
	etag := recW.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header on public GET record")
	}
	cacheCtrl := recW.Header().Get("Cache-Control")
	if !strings.Contains(cacheCtrl, "public") {
		t.Fatalf("expected public Cache-Control header, got %s", cacheCtrl)
	}

	// 2. Subsequent GET with If-None-Match matching ETag -> 304 Not Modified
	req304 := httptest.NewRequest(http.MethodGet, "/api/v1/content/articles/"+id, nil)
	req304.Header.Set("If-None-Match", etag)
	rec304W := httptest.NewRecorder()
	pubHandler.ServeHTTP(rec304W, req304)

	if rec304W.Code != http.StatusNotModified {
		t.Fatalf("expected 304 Not Modified, got %d", rec304W.Code)
	}
	if rec304W.Body.Len() > 0 {
		t.Fatalf("304 response must have empty body, got %s", rec304W.Body.String())
	}

	// 3. Initial GET /api/v1/content/articles (List)
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/content/articles", nil)
	listW := httptest.NewRecorder()
	pubHandler.ServeHTTP(listW, listReq)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", listW.Code)
	}
	listETag := listW.Header().Get("ETag")
	if listETag == "" {
		t.Fatal("expected ETag header on public GET list")
	}

	// 4. Subsequent GET List with If-None-Match -> 304 Not Modified
	list304Req := httptest.NewRequest(http.MethodGet, "/api/v1/content/articles", nil)
	list304Req.Header.Set("If-None-Match", listETag)
	list304W := httptest.NewRecorder()
	pubHandler.ServeHTTP(list304W, list304Req)

	if list304W.Code != http.StatusNotModified {
		t.Fatalf("expected 304 Not Modified for list, got %d", list304W.Code)
	}
}

func TestPublicAPIListETagChangesWhenLowerVersionRecordUpdated(t *testing.T) {
	service, types := newServiceFor(t, publicArticlesYAML)
	adminHandler := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})

	r1, res1 := call(t, adminHandler, http.MethodPost, AdminAPIPath+"/articles", `{"title":"First v1"}`)
	if r1.Code != http.StatusCreated {
		t.Fatalf("seed 1: %d", r1.Code)
	}
	id1 := res1["data"].(map[string]any)["id"].(string)
	for _, title := range []string{"First v2", "First v3", "First v4"} {
		up, _ := call(t, adminHandler, http.MethodPut, AdminAPIPath+"/articles/"+id1, `{"title":"`+title+`"}`)
		if up.Code != http.StatusOK {
			t.Fatalf("bump id1: %d", up.Code)
		}
	}

	r2, res2 := call(t, adminHandler, http.MethodPost, AdminAPIPath+"/articles", `{"title":"Second v1"}`)
	if r2.Code != http.StatusCreated {
		t.Fatalf("seed 2: %d", r2.Code)
	}
	id2 := res2["data"].(map[string]any)["id"].(string)

	pubHandler := baasPublicMux(PublicAPI{Service: service, Types: types})
	list1W := httptest.NewRecorder()
	pubHandler.ServeHTTP(list1W, httptest.NewRequest(http.MethodGet, "/api/v1/content/articles", nil))
	if list1W.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", list1W.Code)
	}
	etag1 := list1W.Header().Get("ETag")
	if etag1 == "" {
		t.Fatal("expected ETag on first list")
	}

	// Update id2 (version 1 -> 2, still strictly below id1's version 4).
	up2, _ := call(t, adminHandler, http.MethodPut, AdminAPIPath+"/articles/"+id2, `{"title":"Second v2"}`)
	if up2.Code != http.StatusOK {
		t.Fatalf("update id2: %d", up2.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/content/articles", nil)
	req2.Header.Set("If-None-Match", etag1)
	list2W := httptest.NewRecorder()
	pubHandler.ServeHTTP(list2W, req2)
	if list2W.Code != http.StatusOK {
		t.Fatalf("expected 200 OK after editing lower-version record, got %d", list2W.Code)
	}
	etag2 := list2W.Header().Get("ETag")
	if etag2 == "" || etag2 == etag1 {
		t.Fatalf("expected new ETag after editing record, got %q (prev %q)", etag2, etag1)
	}
	if !strings.Contains(list2W.Body.String(), "Second v2") {
		t.Fatalf("expected updated body in list response, got %s", list2W.Body.String())
	}
}

type barrierRepo struct {
	collections.Repository
	entered chan struct{}
	release chan struct{}
}

func (b *barrierRepo) GetRecord(ctx context.Context, collectionID, id string) (collections.Record, error) {
	b.entered <- struct{}{}
	<-b.release
	return b.Repository.GetRecord(ctx, collectionID, id)
}

func TestPublicAPIGetTenantIsolationSingleflightAndETag(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "tenant_etag.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	def := Definition{
		ID:     "settings",
		Label:  "Settings",
		Public: true,
		Fields: []Field{{ID: "brand", Label: "Brand", Kind: "text", Required: true}},
	}
	schemas, err := Schemas([]Definition{def})
	if err != nil {
		t.Fatal(err)
	}

	scopeA, err := tenant.NewScope("tenant_a", "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	scopeB, err := tenant.NewScope("tenant_b", "tenant-b")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, tid := range []string{"tenant_a", "tenant_b"} {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO tenants (tenant_id, state, default_locale, timezone, created_at, updated_at) VALUES (?, 'active', 'en', 'UTC', ?, ?)`,
			tid, now, now,
		); err != nil {
			t.Fatal(err)
		}
	}

	store := sqlstore.Store{DB: db, Dialect: database.SQLite, RequireTenantScope: true}
	baseSvc := collections.Service{Repository: store}
	for _, sc := range []tenant.Scope{scopeA, scopeB} {
		tctx := tenant.WithScope(ctx, sc)
		if err := baseSvc.SyncSchemas(tctx, schemas); err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []struct {
		tenantID string
		brand    string
	}{
		{"tenant_a", "Brand Alpha"},
		{"tenant_b", "Brand Beta"},
	} {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO hydro_collection_records (tenant_id, collection_id, id, data_json, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			row.tenantID, "settings", "singleton_1", `{"brand":"`+row.brand+`"}`, 1, now, now,
		); err != nil {
			t.Fatal(err)
		}
	}

	repo := &barrierRepo{
		Repository: store,
		entered:    make(chan struct{}, 2),
		release:    make(chan struct{}),
	}
	pubHandler := baasPublicMux(PublicAPI{
		Service: collections.Service{Repository: repo},
		Types:   []Definition{def},
	})

	// Fire two concurrent GET requests for the SAME record ID ("singleton_1") across two tenants.
	// If singleflight omits tenantID, both callers would block on 1 DB read and leak Brand Alpha or Brand Beta.
	type respResult struct {
		code  int
		etag  string
		brand string
	}
	results := make([]respResult, 2)
	var wg sync.WaitGroup
	for idx, sc := range []tenant.Scope{scopeA, scopeB} {
		wg.Add(1)
		go func(i int, scope tenant.Scope) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/content/settings/singleton_1", nil)
			req = req.WithContext(tenant.WithScope(req.Context(), scope))
			w := httptest.NewRecorder()
			pubHandler.ServeHTTP(w, req)
			var body struct {
				Data map[string]any `json:"data"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &body)
			brand, _ := body.Data["brand"].(string)
			results[i] = respResult{code: w.Code, etag: w.Header().Get("ETag"), brand: brand}
		}(idx, sc)
	}

	// Both requests must independently enter GetRecord (not coalesce across tenants).
	for i := 0; i < 2; i++ {
		select {
		case <-repo.entered:
		case <-time.After(2 * time.Second):
			close(repo.release)
			t.Fatal("expected two separate GetRecord calls across tenants; singleflight coalesced cross-tenant!")
		}
	}
	close(repo.release)
	wg.Wait()

	if results[0].code != http.StatusOK || results[1].code != http.StatusOK {
		t.Fatalf("unexpected status codes: %+v", results)
	}
	if results[0].brand != "Brand Alpha" || results[1].brand != "Brand Beta" {
		t.Fatalf("cross-tenant singleflight leak: got %+v", results)
	}
	if results[0].etag == "" || results[1].etag == "" || results[0].etag == results[1].etag {
		t.Fatalf("expected distinct ETags per tenant, got %+v", results)
	}
}

func TestPublicAPIAccessRulesPrivateCacheControl(t *testing.T) {
	const memberNotesYAML = `id: notes
label: Notes
public: true
access_rules:
  list: "member"
  read: "member"
fields:
  - {id: body, label: Body, kind: text, required: true}
`
	service, types := newServiceFor(t, memberNotesYAML)
	adminHandler := adminMux(AdminAPI{Service: service, Types: types, Authorize: allowAll})
	r, res := call(t, adminHandler, http.MethodPost, AdminAPIPath+"/notes", `{"body":"Secret member note"}`)
	if r.Code != http.StatusCreated {
		t.Fatalf("seed note: %d", r.Code)
	}
	id := res["data"].(map[string]any)["id"].(string)

	pubHandler := baasPublicMux(PublicAPI{
		Service:          service,
		Types:            types,
		ResolvePrincipal: testPrincipalResolver,
	})

	member := Principal{ID: "mem_1", Email: "m@example.com", Role: "member", IsMember: true}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/content/notes", nil)
	listReq = listReq.WithContext(withTestPrincipal(listReq.Context(), member))
	listW := httptest.NewRecorder()
	pubHandler.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("member list expected 200, got %d (%s)", listW.Code, listW.Body.String())
	}
	if got := listW.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Fatalf("expected Cache-Control 'private, no-cache' on restricted list, got %q", got)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/content/notes/"+id, nil)
	getReq = getReq.WithContext(withTestPrincipal(getReq.Context(), member))
	getW := httptest.NewRecorder()
	pubHandler.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("member get expected 200, got %d (%s)", getW.Code, getW.Body.String())
	}
	if got := getW.Header().Get("Cache-Control"); got != "private, no-cache" {
		t.Fatalf("expected Cache-Control 'private, no-cache' on restricted get, got %q", got)
	}
}
