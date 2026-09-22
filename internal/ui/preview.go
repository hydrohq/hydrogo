package ui

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
)

const MaxPreviewPayloadBytes int64 = 256 << 10

// PreviewRenderer must validate the decoded block contract and sanitize any
// rich text before returning template.HTML. The HTTP boundary never accepts
// caller-provided HTML directly.
type PreviewRenderer func([]byte) (template.HTML, error)

// PreviewHandler is the server-side half of the generated Editor.js live
// preview. Authentication, tenant scope and CSRF are intentionally supplied
// by the owning module's protected route middleware.
func PreviewHandler(renderer PreviewRenderer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if renderer == nil {
			http.Error(w, "preview unavailable", http.StatusServiceUnavailable)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, MaxPreviewPayloadBytes+1))
		if err != nil {
			http.Error(w, "preview unavailable", http.StatusBadRequest)
			return
		}
		if int64(len(body)) > MaxPreviewPayloadBytes {
			http.Error(w, "preview payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		var document json.RawMessage
		if err := json.Unmarshal(body, &document); err != nil || len(document) == 0 {
			http.Error(w, "invalid preview document", http.StatusBadRequest)
			return
		}
		markup, err := renderer(document)
		if err != nil {
			if errors.Is(err, errInvalidPreviewDocument) {
				http.Error(w, "invalid preview document", http.StatusBadRequest)
				return
			}
			http.Error(w, "preview unavailable", http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(markup))
	})
}

var errInvalidPreviewDocument = errors.New("invalid preview document")

// InvalidPreviewDocument lets an Edition renderer classify schema failures
// without exposing internal validation details to the editor.
func InvalidPreviewDocument() error { return errInvalidPreviewDocument }
