package security

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
)

const requestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

// RequestIDFromContext returns the server-generated identifier for a request.
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

// RequestIDMiddleware creates a fresh correlation identifier for every request.
// Client-supplied identifiers are intentionally not trusted or propagated.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID, err := newRequestID()
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set(requestIDHeader, requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	id := hex.EncodeToString(raw[:])
	if id == "" {
		return "", errors.New("generated empty request id")
	}
	return id, nil
}
