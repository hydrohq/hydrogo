package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/internal/richtext"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	"github.com/ardaozyildirim/hydrogo/modules/pages/repository/sqlstore"
	"github.com/ardaozyildirim/hydrogo/modules/pages/service"
)

func newTestHandler(t *testing.T) (Handler, http.Handler) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "pages.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	content := service.ContentService{
		Transactions: database.NewTransactor(db),
		Pages:        sqlstore.Store{Dialect: database.SQLite},
		Audit:        auditstore.Store{Dialect: database.SQLite},
		Outbox:       outboxstore.Store{Dialect: database.SQLite},
		PublicCache:  cache.New[pages.Page](4, time.Minute),
		Sanitize:     richtext.NewSanitizer().Sanitize,
	}
	handler := Handler{DB: db, Content: content, DefaultLocale: "tr", PreviewSigner: &preview.Signer{Key: bytes.Repeat([]byte("k"), 32)}}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	return handler, mux
}

func editorContext(role auth.Role) context.Context {
	return auth.WithUser(context.Background(), &auth.User{ID: "user-1", Email: "editor@example.test", Role: role, Active: true})
}

type envelope struct {
	Data     json.RawMessage   `json:"data"`
	Metadata json.RawMessage   `json:"metadata"`
	Code     string            `json:"code"`
	Fields   map[string]string `json:"fields"`
}

func call(t *testing.T, mux http.Handler, ctx context.Context, method, path string, body any) (int, envelope) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader).WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	var out envelope
	if strings.HasPrefix(res.Header().Get("Content-Type"), "application/json") {
		_ = json.Unmarshal(res.Body.Bytes(), &out)
	}
	return res.Code, out
}

func decodePage(t *testing.T, raw json.RawMessage) pages.Page {
	t.Helper()
	var page pages.Page
	if err := json.Unmarshal(raw, &page); err != nil {
		t.Fatalf("decode page: %v (%s)", err, raw)
	}
	return page
}

func TestAdminPagesLifecycle(t *testing.T) {
	_, mux := newTestHandler(t)
	editor := editorContext(auth.RoleEditor)

	status, out := call(t, mux, editor, http.MethodPost, adminBase, map[string]any{"slug": "about", "title": "About", "body": "<p>Hello</p>", "layout_template": "landing"})
	if status != http.StatusCreated {
		t.Fatalf("create status = %d (%s)", status, out.Code)
	}
	created := decodePage(t, out.Data)
	if created.Status != pages.StatusDraft || created.Version != 1 || created.Locale != "tr" || len(created.Blocks) == 0 {
		t.Fatalf("unexpected created page: %+v", created)
	}

	status, out = call(t, mux, editor, http.MethodGet, adminBase+"?search=abo&status=draft", nil)
	if status != http.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	var rows []pageListRow
	if err := json.Unmarshal(out.Data, &rows); err != nil || len(rows) != 1 || rows[0].ID != created.ID {
		t.Fatalf("list rows = %s err=%v", out.Data, err)
	}
	var meta map[string]any
	_ = json.Unmarshal(out.Metadata, &meta)
	if meta["total"] != float64(1) || meta["has_more"] != false {
		t.Fatalf("list metadata = %v", meta)
	}

	status, out = call(t, mux, editor, http.MethodPut, adminBase+"/"+created.ID, map[string]any{"slug": "about", "title": "About us", "body": "<p>Hi</p>", "version": 99})
	if status != http.StatusConflict {
		t.Fatalf("stale update status = %d (%s)", status, out.Code)
	}
	status, out = call(t, mux, editor, http.MethodPut, adminBase+"/"+created.ID, map[string]any{"slug": "about", "title": "About us", "body": "<p>Hi</p>", "version": 1})
	if status != http.StatusOK {
		t.Fatalf("update status = %d (%s)", status, out.Code)
	}
	updated := decodePage(t, out.Data)
	if updated.Version != 2 || updated.Title != "About us" {
		t.Fatalf("unexpected updated page: %+v", updated)
	}

	status, out = call(t, mux, editor, http.MethodPost, adminBase+"/"+created.ID+"/publish", map[string]any{"version": 2})
	if status != http.StatusOK {
		t.Fatalf("publish status = %d (%s)", status, out.Code)
	}
	if published := decodePage(t, out.Data); published.Status != pages.StatusPublished {
		t.Fatalf("page not published: %+v", published)
	}

	status, out = call(t, mux, context.Background(), http.MethodGet, publicBase+"/tr/about", nil)
	if status != http.StatusOK || decodePage(t, out.Data).Title != "About us" {
		t.Fatalf("public page = %d %s", status, out.Data)
	}
	status, out = call(t, mux, context.Background(), http.MethodGet, publicBase+"?locale=tr", nil)
	if status != http.StatusOK || !strings.Contains(string(out.Data), `"slug":"about"`) {
		t.Fatalf("public list = %d %s", status, out.Data)
	}

	status, out = call(t, mux, editor, http.MethodGet, adminBase+"/"+created.ID+"/revisions", nil)
	if status != http.StatusOK {
		t.Fatalf("revisions status = %d", status)
	}
	var revisions []revisionSummary
	if err := json.Unmarshal(out.Data, &revisions); err != nil || len(revisions) != 2 || revisions[0].Revision != 2 {
		t.Fatalf("revisions = %s err=%v", out.Data, err)
	}
	status, out = call(t, mux, editor, http.MethodPost, adminBase+"/"+created.ID+"/revisions/1/restore", map[string]any{"version": 3})
	if status != http.StatusOK {
		t.Fatalf("restore status = %d (%s)", status, out.Code)
	}
	if restored := decodePage(t, out.Data); restored.Title != "About" || restored.Version != 4 {
		t.Fatalf("restored page = %+v", restored)
	}

	status, out = call(t, mux, editor, http.MethodPost, adminBase+"/"+created.ID+"/preview-link", nil)
	if status != http.StatusCreated {
		t.Fatalf("preview link status = %d (%s)", status, out.Code)
	}
	var link struct {
		URL string `json:"url"`
	}
	_ = json.Unmarshal(out.Data, &link)
	status, out = call(t, mux, context.Background(), http.MethodGet, link.URL, nil)
	if status != http.StatusOK || decodePage(t, out.Data).Title != "About" {
		t.Fatalf("preview = %d %s", status, out.Data)
	}
	status, _ = call(t, mux, context.Background(), http.MethodGet, publicBase+"/"+created.ID+"/preview?token=bogus", nil)
	if status != http.StatusNotFound {
		t.Fatalf("bogus preview status = %d", status)
	}
}

func TestAdminPagesPermissionsAndValidation(t *testing.T) {
	_, mux := newTestHandler(t)
	viewer := editorContext(auth.RoleViewer)
	editor := editorContext(auth.RoleEditor)

	if status, out := call(t, mux, viewer, http.MethodPost, adminBase, map[string]any{"slug": "x", "title": "x", "body": "x"}); status != http.StatusForbidden || out.Code != "forbidden" {
		t.Fatalf("viewer create = %d %s", status, out.Code)
	}
	if status, _ := call(t, mux, viewer, http.MethodGet, adminBase, nil); status != http.StatusOK {
		t.Fatalf("viewer list = %d", status)
	}
	if status, _ := call(t, mux, context.Background(), http.MethodGet, adminBase, nil); status != http.StatusForbidden {
		t.Fatalf("anonymous list = %d", status)
	}
	status, out := call(t, mux, editor, http.MethodPost, adminBase, map[string]any{"slug": "", "title": "", "body": ""})
	if status != http.StatusUnprocessableEntity || out.Code != "unprocessable_entity" {
		t.Fatalf("invalid create = %d %s", status, out.Code)
	}
	if status, out := call(t, mux, editor, http.MethodGet, adminBase+"/missing", nil); status != http.StatusNotFound || out.Code != "not_found" {
		t.Fatalf("missing page = %d %s", status, out.Code)
	}
	if status, _ := call(t, mux, editor, http.MethodGet, adminBase+"?limit=0", nil); status != http.StatusBadRequest {
		t.Fatalf("bad limit = %d", status)
	}
	status, out = call(t, mux, editor, http.MethodGet, adminBase+"/blocks", nil)
	if status != http.StatusOK || !strings.Contains(string(out.Data), `"hero"`) {
		t.Fatalf("blocks = %d %s", status, out.Data)
	}
	status, out = call(t, mux, editor, http.MethodGet, adminBase+"/layouts", nil)
	if status != http.StatusOK || !strings.Contains(string(out.Data), `"blank"`) {
		t.Fatalf("layouts = %d %s", status, out.Data)
	}
}
