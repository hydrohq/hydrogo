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
	"github.com/ardaozyildirim/hydrogo/internal/database"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/preview"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/news"
	newsstore "github.com/ardaozyildirim/hydrogo/modules/news/repository/sqlstore"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "news.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	store := newsstore.Store{Dialect: database.SQLite}
	service := news.Service{Transactions: database.NewTransactor(db), Repository: store, Audit: auditstore.Store{Dialect: database.SQLite}, Outbox: outboxstore.Store{Dialect: database.SQLite}}
	handler := Handler{Service: service, DB: db, DefaultLocale: "tr", PreviewSigner: &preview.Signer{Key: bytes.Repeat([]byte("k"), 32)}}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	return mux
}

func userContext(role auth.Role) context.Context {
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

func decodeItem(t *testing.T, raw json.RawMessage) adminItem {
	t.Helper()
	var item adminItem
	if err := json.Unmarshal(raw, &item); err != nil {
		t.Fatalf("decode item: %v (%s)", err, raw)
	}
	return item
}

func TestAdminNewsLifecycle(t *testing.T) {
	mux := newTestHandler(t)
	editor := userContext(auth.RoleEditor)

	status, out := call(t, mux, editor, http.MethodPost, adminBase+"/taxonomy", map[string]any{"id": "launches", "slug": "launches", "kind": "category", "labels": map[string]string{"en": "Launches"}})
	if status != http.StatusOK {
		t.Fatalf("term status = %d (%s)", status, out.Code)
	}

	status, out = call(t, mux, editor, http.MethodPost, adminBase, map[string]any{"slug": "launch", "title": "Launch", "body": "<p>Body</p>", "term_ids": []string{"launches", "unknown"}})
	if status != http.StatusCreated {
		t.Fatalf("create status = %d (%s)", status, out.Code)
	}
	created := decodeItem(t, out.Data)
	if created.Status != news.StatusDraft || created.Version != 1 || len(created.Terms) != 1 || created.Terms[0].ID != "launches" {
		t.Fatalf("unexpected created item: %+v", created)
	}
	if !strings.Contains(strings.Join(created.Transitions, ","), "publish") {
		t.Fatalf("draft should allow publish, got %v", created.Transitions)
	}

	status, out = call(t, mux, editor, http.MethodGet, adminBase+"?status=draft&search=laun", nil)
	if status != http.StatusOK || !strings.Contains(string(out.Data), created.ID) {
		t.Fatalf("list = %d %s", status, out.Data)
	}

	status, out = call(t, mux, editor, http.MethodPut, adminBase+"/"+created.ID, map[string]any{"slug": "launch", "title": "Launch day", "body": "<p>Body</p>", "version": 42})
	if status != http.StatusConflict {
		t.Fatalf("stale update status = %d (%s)", status, out.Code)
	}
	status, out = call(t, mux, editor, http.MethodPut, adminBase+"/"+created.ID, map[string]any{"slug": "launch", "title": "Launch day", "body": "<p>Body</p>", "version": 1})
	if status != http.StatusOK {
		t.Fatalf("update status = %d (%s)", status, out.Code)
	}
	updated := decodeItem(t, out.Data)
	if updated.Version != 2 || updated.Title != "Launch day" {
		t.Fatalf("unexpected updated item: %+v", updated)
	}

	status, out = call(t, mux, editor, http.MethodPost, adminBase+"/"+created.ID+"/schedule", map[string]any{"version": 2, "scheduled_at": time.Now().UTC().Add(time.Hour).Format(time.RFC3339)})
	if status != http.StatusOK {
		t.Fatalf("schedule status = %d (%s)", status, out.Code)
	}
	scheduled := decodeItem(t, out.Data)
	if scheduled.Status != news.StatusScheduled || scheduled.ScheduledAt == nil {
		t.Fatalf("not scheduled: %+v", scheduled)
	}

	status, out = call(t, mux, editor, http.MethodPost, adminBase+"/"+created.ID+"/publish", map[string]any{"version": scheduled.Version})
	if status != http.StatusOK {
		t.Fatalf("publish status = %d (%s)", status, out.Code)
	}
	published := decodeItem(t, out.Data)
	if published.Status != news.StatusPublished {
		t.Fatalf("not published: %+v", published)
	}

	status, out = call(t, mux, context.Background(), http.MethodGet, publicBase+"/tr/launch", nil)
	if status != http.StatusOK || !strings.Contains(string(out.Data), `"title":"Launch day"`) {
		t.Fatalf("public item = %d %s", status, out.Data)
	}
	status, out = call(t, mux, context.Background(), http.MethodGet, publicBase+"?locale=tr", nil)
	if status != http.StatusOK || !strings.Contains(string(out.Data), created.ID) {
		t.Fatalf("public list = %d %s", status, out.Data)
	}

	status, out = call(t, mux, editor, http.MethodPost, adminBase+"/"+created.ID+"/retract", map[string]any{"version": published.Version})
	if status != http.StatusOK || decodeItem(t, out.Data).Status != news.StatusRetracted {
		t.Fatalf("retract = %d %s", status, out.Data)
	}
	if status, _ := call(t, mux, context.Background(), http.MethodGet, publicBase+"/tr/launch", nil); status != http.StatusNotFound {
		t.Fatalf("retracted item still public: %d", status)
	}

	status, out = call(t, mux, editor, http.MethodPost, adminBase+"/"+created.ID+"/preview-link", nil)
	if status != http.StatusCreated {
		t.Fatalf("preview link status = %d (%s)", status, out.Code)
	}
	var link struct {
		URL string `json:"url"`
	}
	_ = json.Unmarshal(out.Data, &link)
	if status, out := call(t, mux, context.Background(), http.MethodGet, link.URL, nil); status != http.StatusOK || !strings.Contains(string(out.Data), created.ID) {
		t.Fatalf("preview = %d %s", status, out.Data)
	}
}

func TestAdminNewsTaxonomyAndPermissions(t *testing.T) {
	mux := newTestHandler(t)
	editor := userContext(auth.RoleEditor)
	viewer := userContext(auth.RoleViewer)

	if status, out := call(t, mux, viewer, http.MethodPost, adminBase, map[string]any{"slug": "x", "title": "x", "body": "x"}); status != http.StatusForbidden || out.Code != "forbidden" {
		t.Fatalf("viewer create = %d %s", status, out.Code)
	}
	if status, out := call(t, mux, editor, http.MethodPost, adminBase+"/taxonomy", map[string]any{"slug": "", "kind": ""}); status != http.StatusUnprocessableEntity || out.Fields["id"] != "required" {
		t.Fatalf("invalid term = %d %v", status, out.Fields)
	}
	for _, id := range []string{"a", "b"} {
		if status, _ := call(t, mux, editor, http.MethodPost, adminBase+"/taxonomy", map[string]any{"id": id, "slug": id, "kind": "tag", "labels": map[string]string{"en": id}}); status != http.StatusOK {
			t.Fatalf("term %s status = %d", id, status)
		}
	}
	status, out := call(t, mux, editor, http.MethodPost, adminBase, map[string]any{"slug": "story", "title": "Story", "body": "<p>x</p>"})
	if status != http.StatusCreated {
		t.Fatalf("create status = %d", status)
	}
	item := decodeItem(t, out.Data)
	status, out = call(t, mux, editor, http.MethodPut, adminBase+"/"+item.ID+"/taxonomy", map[string]any{"term_ids": []string{"b", "a", "b"}})
	if status != http.StatusOK || !strings.Contains(string(out.Data), `"id":"a"`) || !strings.Contains(string(out.Data), `"id":"b"`) {
		t.Fatalf("replace taxonomy = %d %s", status, out.Data)
	}
	if status, out := call(t, mux, editor, http.MethodDelete, adminBase+"/taxonomy/a", nil); status != http.StatusConflict || out.Code != "conflict" {
		t.Fatalf("delete term in use = %d %s", status, out.Code)
	}
	if status, _ := call(t, mux, editor, http.MethodPut, adminBase+"/"+item.ID+"/taxonomy", map[string]any{"term_ids": []string{"b"}}); status != http.StatusOK {
		t.Fatalf("detach term = %d", status)
	}
	if status, _ := call(t, mux, editor, http.MethodDelete, adminBase+"/taxonomy/a", nil); status != http.StatusNoContent {
		t.Fatalf("delete term = %d", status)
	}
	if status, _ := call(t, mux, editor, http.MethodDelete, adminBase+"/taxonomy/a", nil); status != http.StatusNotFound {
		t.Fatalf("delete missing term = %d", status)
	}
	status, out = call(t, mux, editor, http.MethodGet, adminBase+"/taxonomy", nil)
	if status != http.StatusOK || strings.Contains(string(out.Data), `"id":"a"`) || !strings.Contains(string(out.Data), `"id":"b"`) {
		t.Fatalf("catalog after delete = %d %s", status, out.Data)
	}
	if status, out := call(t, mux, editor, http.MethodGet, adminBase+"/missing", nil); status != http.StatusNotFound || out.Code != "not_found" {
		t.Fatalf("missing item = %d %s", status, out.Code)
	}
}
