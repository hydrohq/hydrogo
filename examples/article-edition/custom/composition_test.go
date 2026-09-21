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
	if err != nil { t.Fatal(err) }
	defer db.Close()
	migration, err := os.ReadFile(filepath.Join("..", "generated", "article", "migration.sql"))
	if err != nil { t.Fatal(err) }
	if _, err := db.ExecContext(context.Background(), string(migration)); err != nil { t.Fatal(err) }
	handler := NewDemoHandler(db)
	body, _ := json.Marshal(map[string]string{"id": "a-1", "slug": "demo", "status": "published", "title": "Demo"})
	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/api/v1/article", bytes.NewReader(body)))
	if post.Code != http.StatusCreated { t.Fatalf("POST status = %d body=%s", post.Code, post.Body.String()) }
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/v1/article/a-1", nil))
	if get.Code != http.StatusOK { t.Fatalf("GET status = %d body=%s", get.Code, get.Body.String()) }
}

