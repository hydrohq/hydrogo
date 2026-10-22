// Package bffproxy provides a deliberately small server-side BFF adapter.
//
// It is a reference building block for a separately deployed web frontend. A
// deployment supplies the session and CSRF implementations; this package
// keeps browser credentials out of Core requests and prevents Core cookies
// from leaking back to the browser.
package bffproxy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/security/bffpolicy"
)

var (
	ErrMissingSession = errors.New("bff session is not authenticated")
	ErrMissingCSRF    = errors.New("bff csrf validator is required for mutation")
	ErrResponseLarge  = errors.New("bff upstream response exceeds configured limit")
)

// SessionResolver maps the browser's BFF session to a server-side Core
// session cookie. It must never return the browser Cookie header verbatim.
type SessionResolver interface {
	CoreCookie(ctx context.Context, r *http.Request) (string, error)
}

// CSRFValidator protects the browser-to-BFF boundary for unsafe methods.
type CSRFValidator interface {
	Validate(ctx context.Context, r *http.Request) error
}

// Handler proxies authenticated versioned API requests to Core. Login and
// session creation remain deployment-owned operations: after a successful
// server-side login, the deployment stores the Core cookie behind its BFF
// session and then routes authenticated calls through this handler.
type Handler struct {
	CoreOrigin       *url.URL
	Client           *http.Client
	Sessions         SessionResolver
	CSRF             CSRFValidator
	MaxResponseBytes int64
	MaxRequestBytes  int64
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if isUnsafe(r.Method) {
		if h.CSRF == nil {
			http.Error(w, "csrf validation unavailable", http.StatusInternalServerError)
			return
		}
		if err := h.CSRF.Validate(r.Context(), r); err != nil {
			http.Error(w, "csrf validation failed", http.StatusForbidden)
			return
		}
	}
	if h.Sessions == nil {
		http.Error(w, "bff session unavailable", http.StatusInternalServerError)
		return
	}
	coreCookie, err := h.Sessions.CoreCookie(r.Context(), r)
	if err != nil {
		http.Error(w, "bff session unavailable", http.StatusUnauthorized)
		return
	}
	if strings.TrimSpace(coreCookie) == "" {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	coreRequest, err := bffpolicy.BuildCoreRequest(r, h.CoreOrigin, coreCookie)
	if err != nil {
		http.Error(w, "request rejected", http.StatusBadRequest)
		return
	}
	maxRequest := h.MaxRequestBytes
	if maxRequest <= 0 {
		maxRequest = 1 << 20
	}
	if coreRequest.Body != nil {
		if coreRequest.ContentLength > maxRequest {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		body, readErr := io.ReadAll(io.LimitReader(coreRequest.Body, maxRequest+1))
		_ = coreRequest.Body.Close()
		if readErr != nil {
			http.Error(w, "request body unavailable", http.StatusBadRequest)
			return
		}
		if int64(len(body)) > maxRequest {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		coreRequest.Body = io.NopCloser(bytes.NewReader(body))
		coreRequest.ContentLength = int64(len(body))
		coreRequest.Header.Set("Content-Length", stringSize(len(body)))
	}
	client := h.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(coreRequest) // #nosec G704 -- the upstream is the operator-configured core address; only the request path is forwarded
	if err != nil {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()

	max := h.MaxResponseBytes
	if max <= 0 {
		max = 1 << 20
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, max+1))
	if err != nil {
		http.Error(w, "upstream response unavailable", http.StatusBadGateway)
		return
	}
	if int64(len(body)) > max {
		http.Error(w, "upstream response too large", http.StatusBadGateway)
		return
	}
	copyResponseHeaders(w.Header(), response.Header)
	w.WriteHeader(response.StatusCode)
	_, _ = w.Write(body)
}

func stringSize(size int) string {
	// strconv.FormatInt is kept behind this helper so the request-boundary code
	// has one obvious conversion site.
	return strconv.FormatInt(int64(size), 10)
}

func isUnsafe(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}

func copyResponseHeaders(dst, src http.Header) {
	for _, name := range []string{
		"Content-Type", "Content-Length", "Content-Encoding", "Cache-Control",
		"ETag", "Last-Modified", "Vary", "Retry-After", "Location",
	} {
		for _, value := range src.Values(name) {
			dst.Add(name, value)
		}
	}
}
