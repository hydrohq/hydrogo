package custom

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	generatedadmin "github.com/ardaozyildirim/hydrogo/examples/project-edition/generated"
	generatedmilestone "github.com/ardaozyildirim/hydrogo/examples/project-edition/generated/milestone"
	generated "github.com/ardaozyildirim/hydrogo/examples/project-edition/generated/project"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	_ "modernc.org/sqlite"
)

func TestDemoCompositionWiresGeneratedCRUD(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	migration, err := os.ReadFile(filepath.Join("..", "generated", "project", "migration.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), string(migration)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(context.Background(), generatedmilestone.MigrationSQL); err != nil {
		t.Fatalf("apply generated milestone migration: %v", err)
	}
	handler := NewDemoHandler(db)
	body, _ := json.Marshal(map[string]string{"id": "p-1", "project_code": "P-1", "status": "active", "title": "Demo"})
	server := httptest.NewServer(handler)
	defer server.Close()
	adminIndex, err := http.Get(server.URL + "/admin/resources")
	if err != nil {
		t.Fatal(err)
	}
	adminIndexBody, _ := io.ReadAll(adminIndex.Body)
	_ = adminIndex.Body.Close()
	if adminIndex.StatusCode != http.StatusOK || !bytes.Contains(adminIndexBody, []byte("/admin/resources/project")) || !bytes.Contains(adminIndexBody, []byte("/admin/resources/milestone")) || !bytes.Contains(adminIndexBody, []byte(`class="collection-tile"`)) || !bytes.Contains(adminIndexBody, []byte("</html>")) {
		t.Fatalf("generated resource missing from Admin index: status=%d", adminIndex.StatusCode)
	}
	milestoneCreate, err := http.Post(server.URL+"/api/v1/milestone", "application/json", strings.NewReader(`{"id":"m-1","status":"planned","title":"Golden path"}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = milestoneCreate.Body.Close()
	if milestoneCreate.StatusCode != http.StatusCreated {
		t.Fatalf("generated milestone API create status=%d", milestoneCreate.StatusCode)
	}
	milestoneList, err := http.Get(server.URL + "/admin/resources/milestone")
	if err != nil {
		t.Fatal(err)
	}
	milestoneListBody, _ := io.ReadAll(milestoneList.Body)
	_ = milestoneList.Body.Close()
	if milestoneList.StatusCode != http.StatusOK || !bytes.Contains(milestoneListBody, []byte("Golden path")) {
		t.Fatalf("new Blueprint resource Admin list status=%d body=%s", milestoneList.StatusCode, milestoneListBody)
	}
	post, err := http.Post(server.URL+"/api/v1/project", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if post.StatusCode != http.StatusCreated {
		t.Fatalf("POST status = %d", post.StatusCode)
	}
	_ = post.Body.Close()
	get, err := http.Get(server.URL + "/api/v1/project/p-1")
	if err != nil {
		t.Fatal(err)
	}
	if get.StatusCode != http.StatusOK {
		t.Fatalf("GET status = %d", get.StatusCode)
	}
	_ = get.Body.Close()
	listAPI, err := http.Get(server.URL + "/api/v1/project?limit=100&offset=0")
	if err != nil {
		t.Fatal(err)
	}
	listAPIJSON, _ := io.ReadAll(listAPI.Body)
	_ = listAPI.Body.Close()
	if listAPI.StatusCode != http.StatusOK || !bytes.Contains(listAPIJSON, []byte("Demo")) {
		t.Fatalf("generated list API status=%d body=%s", listAPI.StatusCode, listAPIJSON)
	}
	adminList, err := http.Get(server.URL + "/admin/resources/project")
	if err != nil {
		t.Fatal(err)
	}
	adminBody, _ := io.ReadAll(adminList.Body)
	_ = adminList.Body.Close()
	if adminList.StatusCode != http.StatusOK || !bytes.Contains(adminBody, []byte("Demo")) || !bytes.Contains(adminBody, []byte("</html>")) {
		body := string(adminBody)
		index := strings.Index(body, "admin-resource")
		if index < 0 {
			index = max(0, len(body)-500)
		}
		t.Fatalf("generated Admin list status=%d body=%s", adminList.StatusCode, body[index:min(len(body), index+500)])
	}
	adminForm, err := http.Get(server.URL + "/admin/resources/project/new")
	if err != nil {
		t.Fatal(err)
	}
	adminFormBody, _ := io.ReadAll(adminForm.Body)
	_ = adminForm.Body.Close()
	if adminForm.StatusCode != http.StatusOK || !bytes.Contains(adminFormBody, []byte(`class="studio-form"`)) || !bytes.Contains(adminFormBody, []byte("</html>")) {
		t.Fatalf("generated Admin form layout missing: status=%d body=%s", adminForm.StatusCode, adminFormBody)
	}
	form := url.Values{"id": {"p-admin"}, "project_code": {"P-ADMIN"}, "status": {"active"}, "title": {"From Admin"}, "_csrf": {"edition-demo-csrf"}}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	createRequest, _ := http.NewRequest(http.MethodPost, server.URL+"/admin/resources/project/new", strings.NewReader(form.Encode()))
	createRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	created, err := client.Do(createRequest)
	if err != nil {
		t.Fatal(err)
	}
	createdBody, _ := io.ReadAll(created.Body)
	_ = created.Body.Close()
	if created.StatusCode != http.StatusSeeOther || created.Header.Get("Location") != "/admin/resources/project" {
		t.Fatalf("generated Admin create status=%d location=%q body=%s", created.StatusCode, created.Header.Get("Location"), createdBody)
	}
	adminList, err = http.Get(server.URL + created.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	adminBody, _ = io.ReadAll(adminList.Body)
	_ = adminList.Body.Close()
	if adminList.StatusCode != http.StatusOK || !bytes.Contains(adminBody, []byte("From Admin")) || !bytes.Contains(adminBody, []byte("</html>")) {
		t.Fatalf("generated Admin list after create status=%d body=%s", adminList.StatusCode, adminBody)
	}
	detail, err := http.Get(server.URL + "/admin/resources/project/p-admin")
	if err != nil {
		t.Fatal(err)
	}
	detailBody, _ := io.ReadAll(detail.Body)
	_ = detail.Body.Close()
	if detail.StatusCode != http.StatusOK || !bytes.Contains(detailBody, []byte("From Admin")) || !bytes.Contains(detailBody, []byte("</html>")) {
		t.Fatalf("generated Admin detail status=%d body=%s", detail.StatusCode, detailBody)
	}
	update := url.Values{"project_code": {"P-ADMIN"}, "status": {"active"}, "title": {"Updated in Admin"}, "_csrf": {"edition-demo-csrf"}}
	updateRequest, _ := http.NewRequest(http.MethodPost, server.URL+"/admin/resources/project/p-admin", strings.NewReader(update.Encode()))
	updateRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updated, err := client.Do(updateRequest)
	if err != nil {
		t.Fatal(err)
	}
	_ = updated.Body.Close()
	if updated.StatusCode != http.StatusSeeOther {
		t.Fatalf("generated Admin update status=%d", updated.StatusCode)
	}
	detail, err = http.Get(server.URL + "/admin/resources/project/p-admin")
	if err != nil {
		t.Fatal(err)
	}
	detailBody, _ = io.ReadAll(detail.Body)
	_ = detail.Body.Close()
	if detail.StatusCode != http.StatusOK || !bytes.Contains(detailBody, []byte("Updated in Admin")) {
		t.Fatalf("generated Admin detail after update status=%d body=%s", detail.StatusCode, detailBody)
	}
	deleteRequest, _ := http.NewRequest(http.MethodPost, server.URL+"/admin/resources/project/p-admin/delete", strings.NewReader("_csrf=edition-demo-csrf"))
	deleteRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deleted, err := client.Do(deleteRequest)
	if err != nil {
		t.Fatal(err)
	}
	_ = deleted.Body.Close()
	if deleted.StatusCode != http.StatusSeeOther {
		t.Fatalf("generated Admin delete status=%d", deleted.StatusCode)
	}
	deletedRecord, err := http.Get(server.URL + "/api/v1/project/p-admin")
	if err != nil {
		t.Fatal(err)
	}
	_ = deletedRecord.Body.Close()
	if deletedRecord.StatusCode != http.StatusNotFound {
		t.Fatalf("generated API record after Admin delete status=%d", deletedRecord.StatusCode)
	}
}

func TestGeneratedPermissionMutationMatrixPreservesSQLiteState(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.ExecContext(context.Background(), generated.MigrationSQL); err != nil {
		t.Fatal(err)
	}
	runGeneratedPermissionMutationMatrix(t, db, "sqlite")
}

func TestPostgresGeneratedPermissionMutationMatrixPreservesState(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	adminDB, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = adminDB.Close() })
	schema := fmt.Sprintf("project_policy_%d", time.Now().UnixNano())
	if _, err := adminDB.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = adminDB.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE") })
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()
	db, err := database.Open(ctx, database.Postgres, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, generated.MigrationSQL); err != nil {
		t.Fatal(err)
	}
	runGeneratedPermissionMutationMatrix(t, db, "postgres")
}

func runGeneratedPermissionMutationMatrix(t *testing.T, db *sql.DB, dialect string) {
	t.Helper()
	policy := &mutationMatrixPolicy{read: true, create: true, update: true, delete: true}
	mux := http.NewServeMux()
	if err := generatedadmin.RegisterGeneratedResources(mux, generatedadmin.GeneratedResourceOptions{
		DB: db, Dialect: dialect,
		PolicyFor: func(string) generatedadmin.GeneratedResourcePolicy { return policy },
	}); err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader(body)))
		return recorder
	}
	if got := request(http.MethodPost, "/api/v1/project", `{"id":"matrix-1","project_code":"M-1","status":"active","title":"Original"}`); got.Code != http.StatusCreated {
		t.Fatalf("fixture create status=%d body=%s", got.Code, got.Body.String())
	}

	cases := []struct {
		name   string
		deny   func()
		allow  func()
		method string
		path   string
		body   string
		status int
	}{
		{name: "read item", deny: func() { policy.read = false }, allow: func() { policy.read = true }, method: http.MethodGet, path: "/api/v1/project/matrix-1", status: http.StatusNotFound},
		{name: "read list", deny: func() { policy.read = false }, allow: func() { policy.read = true }, method: http.MethodGet, path: "/api/v1/project", status: http.StatusForbidden},
		{name: "create", deny: func() { policy.create = false }, allow: func() { policy.create = true }, method: http.MethodPost, path: "/api/v1/project", body: `{"id":"matrix-denied","project_code":"M-X","status":"active","title":"Denied"}`, status: http.StatusForbidden},
		{name: "update", deny: func() { policy.update = false }, allow: func() { policy.update = true }, method: http.MethodPut, path: "/api/v1/project/matrix-1", body: `{"project_code":"M-1","status":"active","title":"Changed"}`, status: http.StatusForbidden},
		{name: "delete", deny: func() { policy.delete = false }, allow: func() { policy.delete = true }, method: http.MethodDelete, path: "/api/v1/project/matrix-1", status: http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.deny()
			if got := request(tc.method, tc.path, tc.body); got.Code != tc.status {
				t.Fatalf("denied request status=%d want=%d body=%s", got.Code, tc.status, got.Body.String())
			}
			tc.allow()
			item := request(http.MethodGet, "/api/v1/project/matrix-1", "")
			var record map[string]any
			if item.Code != http.StatusOK || json.Unmarshal(item.Body.Bytes(), &record) != nil || record["title"] != "Original" {
				t.Fatalf("denied mutation changed persisted record: status=%d body=%s", item.Code, item.Body.String())
			}
			missing := request(http.MethodGet, "/api/v1/project/matrix-denied", "")
			if missing.Code != http.StatusNotFound {
				t.Fatalf("denied create persisted a record: status=%d body=%s", missing.Code, missing.Body.String())
			}
		})
	}
}

type mutationMatrixPolicy struct{ read, create, update, delete bool }

func (p *mutationMatrixPolicy) CanRead(context.Context, string) bool   { return p.read }
func (p *mutationMatrixPolicy) CanCreate(context.Context) bool         { return p.create }
func (p *mutationMatrixPolicy) CanUpdate(context.Context, string) bool { return p.update }
func (p *mutationMatrixPolicy) CanDelete(context.Context, string) bool { return p.delete }

func TestGeneratedResourceRoutesFailClosedBeforeRegisteringAnyRoute(t *testing.T) {
	mux := http.NewServeMux()
	err := generatedadmin.RegisterGeneratedResources(mux, generatedadmin.GeneratedResourceOptions{
		DB: &sql.DB{}, Dialect: "sqlite",
		PolicyFor: func(resourceID string) generatedadmin.GeneratedResourcePolicy {
			if resourceID == "milestone" {
				return demoPolicy{}
			}
			return nil
		},
	})
	if err == nil {
		t.Fatal("missing policy for one generated resource was accepted")
	}
	for _, path := range []string{"/api/v1/milestone", "/api/v1/project"} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Errorf("route %s was partially registered after preflight failure: status=%d", path, response.Code)
		}
	}
}

func TestPostgresEditionGeneratedAdminCRUD(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	ctx := context.Background()
	adminDB, err := database.Open(ctx, database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer adminDB.Close()
	schema := fmt.Sprintf("project_crud_%d", time.Now().UnixNano())
	if _, err := adminDB.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	})

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()
	db, err := database.Open(ctx, database.Postgres, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, generated.MigrationSQL); err != nil {
		t.Fatalf("apply generated Edition migration: %v", err)
	}
	if _, err := db.ExecContext(ctx, generatedmilestone.MigrationSQL); err != nil {
		t.Fatalf("apply generated milestone migration: %v", err)
	}
	server := httptest.NewServer(NewHandler(db, "postgres"))
	defer server.Close()

	postJSON := func(path, body string) *http.Response {
		t.Helper()
		response, err := http.Post(server.URL+path, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	apiCreate := postJSON("/api/v1/project", `{"id":"pg-api","project_code":"PG-API","status":"active","title":"PostgreSQL API record"}`)
	if apiCreate.StatusCode != http.StatusCreated {
		t.Fatalf("generated PostgreSQL API create status=%d", apiCreate.StatusCode)
	}
	_ = apiCreate.Body.Close()
	adminIndex, err := http.Get(server.URL + "/admin/resources")
	if err != nil {
		t.Fatal(err)
	}
	indexBody, _ := io.ReadAll(adminIndex.Body)
	_ = adminIndex.Body.Close()
	if adminIndex.StatusCode != http.StatusOK || !bytes.Contains(indexBody, []byte("/admin/resources/project")) || !bytes.Contains(indexBody, []byte("/admin/resources/milestone")) {
		t.Fatalf("generated collection missing from PostgreSQL Edition Admin index: status=%d", adminIndex.StatusCode)
	}
	milestoneCreate := postJSON("/api/v1/milestone", `{"id":"pg-milestone","status":"planned","title":"PostgreSQL milestone"}`)
	if milestoneCreate.StatusCode != http.StatusCreated {
		t.Fatalf("generated PostgreSQL milestone API create status=%d", milestoneCreate.StatusCode)
	}
	_ = milestoneCreate.Body.Close()
	milestoneList, err := http.Get(server.URL + "/admin/resources/milestone?search=PostgreSQL")
	if err != nil {
		t.Fatal(err)
	}
	milestoneListBody, _ := io.ReadAll(milestoneList.Body)
	_ = milestoneList.Body.Close()
	if milestoneList.StatusCode != http.StatusOK || !bytes.Contains(milestoneListBody, []byte("PostgreSQL milestone")) {
		t.Fatalf("generated PostgreSQL milestone Admin list status=%d body=%s", milestoneList.StatusCode, milestoneListBody)
	}

	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	create := url.Values{"id": {"pg-admin"}, "project_code": {"PG-ADMIN"}, "status": {"active"}, "title": {"PostgreSQL Admin record"}, "_csrf": {"edition-demo-csrf"}}
	createResponse, err := client.PostForm(server.URL+"/admin/resources/project/new", create)
	if err != nil {
		t.Fatal(err)
	}
	_ = createResponse.Body.Close()
	if createResponse.StatusCode != http.StatusSeeOther {
		t.Fatalf("generated PostgreSQL Admin create status=%d", createResponse.StatusCode)
	}
	list, err := http.Get(server.URL + "/admin/resources/project?search=PostgreSQL")
	if err != nil {
		t.Fatal(err)
	}
	listBody, _ := io.ReadAll(list.Body)
	_ = list.Body.Close()
	if list.StatusCode != http.StatusOK || !bytes.Contains(listBody, []byte("PostgreSQL Admin record")) {
		t.Fatalf("generated PostgreSQL Admin list status=%d body=%s", list.StatusCode, listBody)
	}

	update := url.Values{"project_code": {"PG-ADMIN"}, "status": {"active"}, "title": {"Updated PostgreSQL Admin record"}, "_csrf": {"edition-demo-csrf"}}
	updateResponse, err := client.PostForm(server.URL+"/admin/resources/project/pg-admin", update)
	if err != nil {
		t.Fatal(err)
	}
	_ = updateResponse.Body.Close()
	if updateResponse.StatusCode != http.StatusSeeOther {
		t.Fatalf("generated PostgreSQL Admin update status=%d", updateResponse.StatusCode)
	}
	deleteResponse, err := client.PostForm(server.URL+"/admin/resources/project/pg-admin/delete", url.Values{"_csrf": {"edition-demo-csrf"}})
	if err != nil {
		t.Fatal(err)
	}
	_ = deleteResponse.Body.Close()
	if deleteResponse.StatusCode != http.StatusSeeOther {
		t.Fatalf("generated PostgreSQL Admin delete status=%d", deleteResponse.StatusCode)
	}
	deleted, err := http.Get(server.URL + "/api/v1/project/pg-admin")
	if err != nil {
		t.Fatal(err)
	}
	_ = deleted.Body.Close()
	if deleted.StatusCode != http.StatusNotFound {
		t.Fatalf("generated PostgreSQL record after Admin delete status=%d", deleted.StatusCode)
	}
}
