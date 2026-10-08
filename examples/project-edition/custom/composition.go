package custom

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"strings"

	generatedadmin "github.com/ardaozyildirim/hydrogo/examples/project-edition/generated"
	adminstudio "github.com/ardaozyildirim/hydrogo/extension/v1/adminstudio"
	staticassets "github.com/ardaozyildirim/hydrogo/static"
	"github.com/microcosm-cc/bluemonday"
)

const demoCSRFToken = "edition-demo-csrf"

const maxContentBlockBytes = 256 << 10

type editorDocument struct {
	Time    int64         `json:"time,omitempty"`
	Blocks  []editorBlock `json:"blocks"`
	Version string        `json:"version,omitempty"`
}

type editorBlock struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type"`
	Data struct {
		Text string `json:"text"`
	} `json:"data"`
}

var editorHTMLPolicy = bluemonday.UGCPolicy()

//go:embed assets/editorjs.umd.js
var editorJS []byte

// NewDemoHandler wires generated persistence and transport for local contract
// tests only. A real Edition must replace demoPolicy with tenant/member-aware
// authorization before exposing the handler.
func NewDemoHandler(db *sql.DB) http.Handler {
	return NewHandler(db, "sqlite")
}

// NewHandler is a demo Edition composition point, not a production auth stack.
// Its permissive callbacks are for the loopback-only fixture server and local
// contract tests; a real Edition must replace them with its identity, CSRF,
// resource visibility, and tenant-aware authorization boundaries.
func NewHandler(db *sql.DB, dialect string) http.Handler {
	mux := http.NewServeMux()
	vendorFS, err := fs.Sub(staticassets.FS, "vendor")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /vendor/{path...}", http.StripPrefix("/vendor/", http.FileServer(http.FS(vendorFS))))
	apiMux := http.NewServeMux()
	if err := generatedadmin.RegisterGeneratedResources(apiMux, generatedadmin.GeneratedResourceOptions{
		DB: db, Dialect: dialect,
		PolicyFor: func(string) generatedadmin.GeneratedResourcePolicy {
			// This permissive fixture is safe only because the example server
			// refuses non-loopback binds; real Editions must resolve permissions.
			return demoPolicy{}
		},
	}); err != nil {
		panic(err)
	}
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := sanitizeContentBlockMutation(w, r); err != nil {
			return
		}
		apiMux.ServeHTTP(w, r)
	})
	mux.Handle("/api/v1/", api)
	mux.HandleFunc("GET /edition/assets/editorjs.umd.js", serveEditorJS)
	mux.Handle("POST /admin/resources/content_block/preview", contentBlockPreviewHandler())
	if err := generatedadmin.RegisterGeneratedAdmin(mux, adminstudio.GeneratedAdminOptions{
		API: api, Authorize: func(*http.Request) bool { return true }, CanReadResource: func(*http.Request, adminstudio.Collection) bool { return true },
		ResourceAccess: func(*http.Request, adminstudio.Collection, string) adminstudio.ResourceAccess {
			return adminstudio.ResourceAccess{Create: true, Update: true, Delete: true}
		},
		CSRFToken: func(*http.Request) string { return "edition-demo-csrf" },
		ValidateCSRF: func(r *http.Request) bool {
			return r.ParseForm() == nil && r.FormValue("_csrf") == "edition-demo-csrf"
		},
		RenderPage: adminstudio.StudioPageRenderer(func(*http.Request) adminstudio.PageIdentity {
			return adminstudio.PageIdentity{UserID: "demo-admin", Email: "admin@example.test", Role: adminstudio.PageRoleAdmin}
		}),
		BlockEditor: adminstudio.BlockEditorConfig{EditorScript: "/edition/assets/editorjs.umd.js", PreviewURL: "/admin/resources/content_block/preview"},
	}); err != nil {
		panic(err)
	}
	return mux
}

func serveEditorJS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(editorJS)
}

func sanitizeContentBlockMutation(w http.ResponseWriter, r *http.Request) error {
	if (r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch) || r.URL.Path != "/api/v1/content_block" && !strings.HasPrefix(r.URL.Path, "/api/v1/content_block/") {
		return nil
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid request", http.StatusRequestEntityTooLarge)
		return err
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return err
	}
	raw, exists := payload["body"]
	if !exists {
		return nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		writeInvalidBlockDocument(w)
		return err
	}
	clean, err := normalizeEditorDocument(value)
	if err != nil {
		writeInvalidBlockDocument(w)
		return err
	}
	cleanJSON, err := json.Marshal(clean)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return err
	}
	payload["body"] = cleanJSON
	cleanBody, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return err
	}
	r.Body = io.NopCloser(bytes.NewReader(cleanBody))
	r.ContentLength = int64(len(cleanBody))
	return nil
}

func writeInvalidBlockDocument(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_, _ = io.WriteString(w, `{"code":"validation_failed","message":"One or more fields are invalid.","fields":{"body":"invalid_blocks"}}`)
}

func normalizeEditorDocument(value string) (string, error) {
	if len(value) == 0 || len(value) > maxContentBlockBytes {
		return "", errors.New("invalid block document size")
	}
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	var document editorDocument
	if err := decoder.Decode(&document); err != nil || len(document.Blocks) == 0 || len(document.Blocks) > 100 {
		return "", errors.New("invalid block document")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return "", errors.New("invalid block document")
	}
	for i := range document.Blocks {
		block := &document.Blocks[i]
		if block.Type != "paragraph" || len(block.Data.Text) > 50000 {
			return "", errors.New("unsupported block")
		}
		block.Data.Text = editorHTMLPolicy.Sanitize(block.Data.Text)
	}
	encoded, err := json.Marshal(document)
	if err != nil || len(encoded) > maxContentBlockBytes {
		return "", errors.New("invalid block document size")
	}
	return string(encoded), nil
}

func contentBlockPreviewHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/x-www-form-urlencoded" {
			http.Error(w, "invalid preview request", http.StatusBadRequest)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxContentBlockBytes*4)
		if err := r.ParseForm(); err != nil {
			var sizeErr *http.MaxBytesError
			if errors.As(err, &sizeErr) {
				http.Error(w, "preview payload too large", http.StatusRequestEntityTooLarge)
			} else {
				http.Error(w, "invalid preview request", http.StatusBadRequest)
			}
			return
		}
		if r.FormValue("_csrf") != demoCSRFToken {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		values := r.Form["body"]
		if len(values) != 1 {
			http.Error(w, "invalid preview request", http.StatusBadRequest)
			return
		}
		clean, err := normalizeEditorDocument(values[0])
		if err != nil {
			http.Error(w, "invalid preview document", http.StatusBadRequest)
			return
		}
		var document editorDocument
		if err := json.Unmarshal([]byte(clean), &document); err != nil {
			http.Error(w, "invalid preview document", http.StatusBadRequest)
			return
		}
		var markup strings.Builder
		for _, block := range document.Blocks {
			fmt.Fprintf(&markup, "<p>%s</p>", block.Data.Text)
		}
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, markup.String())
	})
}

type demoPolicy struct{}

func (demoPolicy) CanRead(context.Context, string) bool   { return true }
func (demoPolicy) CanCreate(context.Context) bool         { return true }
func (demoPolicy) CanUpdate(context.Context, string) bool { return true }
func (demoPolicy) CanDelete(context.Context, string) bool { return true }
