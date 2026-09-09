package security

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var ErrInvalidJSON = errors.New("invalid JSON request")

// DecodeJSONStrict applies the common API input contract: bounded body,
// rejected unknown fields, and exactly one JSON value.
func DecodeJSONStrict(w http.ResponseWriter, r *http.Request, target any, maxBytes int64) error {
	if r == nil || r.Body == nil || maxBytes <= 0 {
		return ErrInvalidJSON
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return ErrInvalidJSON
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return ErrInvalidJSON
	}
	return nil
}
