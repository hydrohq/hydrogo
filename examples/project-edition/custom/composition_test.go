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
	if adminIndex.StatusCode != http.StatusOK || !bytes.Contains(adminIndexBody, []byte("/admin/resources/project")) || !bytes.Contains(adminIndexBody, []byte(`class="collection-tile"`)) || !bytes.Contains(adminIndexBody, []byte("</html>")) {
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
