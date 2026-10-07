package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/a-h/templ"
	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	staticassets "github.com/ardaozyildirim/hydrogo/static"
)

func main() {
	registry, err := adminstudio.NewRegistry([]adminstudio.Collection{{
		ID: "article", Label: "Articles", Group: "Content", TitleField: "title",
		Fields: []adminstudio.Field{
			{ID: "title", Label: "Title", Kind: adminstudio.FieldText, Required: true, VisibleInList: true},
			{ID: "body", Label: "Article body", Kind: adminstudio.FieldText, Widget: "richtext"},
		},
		Actions: []adminstudio.Action{{Label: "Articles", Href: "/admin/resources/article"}},
	}})
	if err != nil {
		panic(err)
	}

	var mu sync.Mutex
	records := map[string]map[string]any{}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/article"), "/"), "/")
		if r.URL.Path == "/api/v1/article" {
			if r.Method == http.MethodPost {
				var record map[string]any
				if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
					writeError(w, http.StatusBadRequest)
					return
				}
				id, _ := record["id"].(string)
				if id == "" {
					id = "article-" + fmt.Sprint(len(records)+1)
				}
				record["id"] = id
				mu.Lock()
				records[id] = record
				mu.Unlock()
				writeJSON(w, http.StatusCreated, clone(record))
				return
			}
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed)
				return
			}
			mu.Lock()
			result := make([]map[string]any, 0, len(records))
			for _, record := range records {
				result = append(result, clone(record))
			}
			mu.Unlock()
			writeJSON(w, http.StatusOK, result)
			return
		}
		if len(parts) != 1 || parts[0] == "" {
			writeError(w, http.StatusNotFound)
			return
		}
		id := parts[0]
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case http.MethodGet:
			record, ok := records[id]
			if !ok {
				writeError(w, http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, clone(record))
		case http.MethodPost, http.MethodPut:
			var record map[string]any
			if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
				writeError(w, http.StatusBadRequest)
				return
			}
			if id == "new" || id == "" || r.Method == http.MethodPost {
				if _, exists := records[id]; exists || id == "" || id == "new" {
					id = "article-" + fmt.Sprint(len(records)+1)
				}
				record["id"] = id
				records[id] = record
				writeJSON(w, http.StatusCreated, clone(record))
				return
			}
			record["id"] = id
			records[id] = record
			writeJSON(w, http.StatusOK, clone(record))
		case http.MethodDelete:
			delete(records, id)
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, http.StatusMethodNotAllowed)
		}
	})

	mux := http.NewServeMux()
	vendorFS, err := fs.Sub(staticassets.FS, "vendor")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /vendor/{path...}", http.StripPrefix("/vendor/", http.FileServer(http.FS(vendorFS))))
	if err := (adminstudio.GeneratedAdminHandler{Registry: registry, Options: adminstudio.GeneratedAdminOptions{
		API:             api,
		Authorize:       func(*http.Request) bool { return true },
		CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		CSRFToken:       func(*http.Request) string { return "generated-richtext-e2e-csrf" },
		ValidateCSRF: func(r *http.Request) bool {
			return r.ParseForm() == nil && r.FormValue("_csrf") == "generated-richtext-e2e-csrf"
		},
		RenderPage: func(w http.ResponseWriter, r *http.Request, title string, body templ.Component) error {
			if err := ui.AdminStartForRequest(w, r, title, "", "generated"); err != nil {
				return err
			}
			if err := body.Render(r.Context(), w); err != nil {
				return err
			}
			return ui.AdminEnd(w)
		},
	}}).Register(mux); err != nil {
		panic(err)
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	address := os.Getenv("HYDRO_E2E_ADDRESS")
	if address == "" {
		address = "127.0.0.1:8093"
	}
	if err := http.ListenAndServe(address, mux); err != nil {
		panic(err)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func clone(record map[string]any) map[string]any {
	encoded, _ := json.Marshal(record)
	var copy map[string]any
	_ = json.Unmarshal(encoded, &copy)
	return copy
}
