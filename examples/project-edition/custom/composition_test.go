package custom

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
	if adminIndex.StatusCode != http.StatusOK || !bytes.Contains(adminIndexBody, []byte("/admin/resources/project")) {
		t.Fatalf("generated resource missing from Admin index: status=%d", adminIndex.StatusCode)
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
	if adminList.StatusCode != http.StatusOK || !bytes.Contains(adminBody, []byte("Demo")) {
		body := string(adminBody)
		index := strings.Index(body, "admin-resource")
		if index < 0 {
			index = max(0, len(body)-500)
		}
		t.Fatalf("generated Admin list status=%d body=%s", adminList.StatusCode, body[index:min(len(body), index+500)])
	}
	form := url.Values{"id": {"p-admin"}, "project_code": {"P-ADMIN"}, "status": {"active"}, "title": {"From Admin"}, "_csrf": {"edition-demo-csrf"}}
	created, err := http.PostForm(server.URL+"/admin/resources/project/new", form)
	if err != nil {
		t.Fatal(err)
	}
	_ = created.Body.Close()
	if created.StatusCode != http.StatusOK {
		t.Fatalf("generated Admin create status=%d", created.StatusCode)
	}
}
