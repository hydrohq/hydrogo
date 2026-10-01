package api

import (
	"encoding/json"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/security"
)

// ResponseData is the standard shape for JSON responses containing data.
// It allows for metadata (like pagination) to be returned alongside the data.
type ResponseData struct {
	Data     any `json:"data"`
	Metadata any `json:"metadata,omitempty"`
}

// PaginationMeta holds standard cursor-based pagination metadata.
type PaginationMeta struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
	Total      int    `json:"total,omitempty"`
}

// WriteJSON writes a standard JSON response with HTTP 200 OK (or provided status).
func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// WriteData writes a JSON response wrapped in the standard {"data": ...} shape.
func WriteData(w http.ResponseWriter, status int, data any, metadata any) {
	WriteJSON(w, status, ResponseData{Data: data, Metadata: metadata})
}

// WriteError writes a structured API error, allowing for field-level validation errors.
func WriteError(w http.ResponseWriter, r *http.Request, status int, message string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(security.ErrorResponse{
		Code:      security.ErrorCode(status),
		Message:   message,
		Fields:    fields,
		RequestID: security.RequestIDFromContext(r.Context()),
	})
}
