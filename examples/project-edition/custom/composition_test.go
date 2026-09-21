package custom

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
}
