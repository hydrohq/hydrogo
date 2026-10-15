package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/security"
)

// ResponseData is the standard shape for JSON responses containing data.
// It allows for metadata (like pagination) to be returned alongside the data.
type ResponseData struct {
	Data     any `json:"data"`
	Metadata any `json:"metadata,omitempty"`
}

// PaginationMeta describes a page of a list. Offset lists fill Limit/Offset
// (and Total when it is cheap to know); cursor lists fill NextCursor.
type PaginationMeta struct {
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
	Total      int    `json:"total,omitempty"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// DefaultMaxBodyBytes bounds JSON request bodies that carry no file content.
const DefaultMaxBodyBytes int64 = 1 << 20

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

// WriteList writes a page of items with pagination metadata.
func WriteList(w http.ResponseWriter, items any, page PaginationMeta) {
	WriteData(w, http.StatusOK, items, page)
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

// WriteValidation answers 422 with per-field codes ("required", "invalid",
// "taken", ...) so a client can attach them to form inputs.
func WriteValidation(w http.ResponseWriter, r *http.Request, fields map[string]string) {
	WriteError(w, r, http.StatusUnprocessableEntity, "validation failed", fields)
}

// DecodeJSON reads a bounded JSON object into dst. On failure it answers the
// request (400 or 413) and returns false, so handlers can simply return.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) bool {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			WriteError(w, r, http.StatusRequestEntityTooLarge, "request body too large", nil)
			return false
		}
		WriteError(w, r, http.StatusBadRequest, "invalid JSON body", nil)
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		WriteError(w, r, http.StatusBadRequest, "invalid JSON body", nil)
		return false
	}
	return true
}
