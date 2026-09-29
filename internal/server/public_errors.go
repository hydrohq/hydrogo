package server

import (
	"context"
	"io"
	"net/http"
	"strings"
)

type PublicErrorPageRenderer func(context.Context, int) ([]byte, bool)

const maxOriginalErrorBody = 64 << 10

func PublicErrorPageMiddleware(render PublicErrorPageRenderer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if render == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !publicHTMLPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			capture := &publicErrorWriter{ResponseWriter: w, request: r, render: render}
			next.ServeHTTP(capture, r)
			capture.finish()
		})
	}
}

func publicHTMLPath(path string) bool {
	for _, prefix := range []string{"/api", "/admin", "/auth", "/healthz", "/readyz", "/.well-known"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return false
		}
	}
	return true
}

type publicErrorWriter struct {
	http.ResponseWriter
	request     *http.Request
	render      PublicErrorPageRenderer
	status      int
	wroteHeader bool
	capturing   bool
	body        []byte
}

func (w *publicErrorWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *publicErrorWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader, w.status = true, status
	if status == http.StatusNotFound || status == http.StatusInternalServerError {
		w.capturing = true
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *publicErrorWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.capturing {
		remaining := maxOriginalErrorBody - len(w.body)
		if remaining > len(body) {
			remaining = len(body)
		}
		if remaining > 0 {
			w.body = append(w.body, body[:remaining]...)
		}
		return len(body), nil
	}
	return w.ResponseWriter.Write(body)
}

func (w *publicErrorWriter) Flush() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.capturing {
		w.ResponseWriter.WriteHeader(w.status)
		_, _ = w.ResponseWriter.Write(w.body)
		w.capturing = false
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *publicErrorWriter) finish() {
	if !w.capturing {
		return
	}
	if body, ok := w.render(w.request.Context(), w.status); ok {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Del("Content-Length")
		w.ResponseWriter.WriteHeader(w.status)
		_, _ = w.ResponseWriter.Write(body)
		return
	}
	w.ResponseWriter.WriteHeader(w.status)
	_, _ = io.WriteString(w.ResponseWriter, string(w.body))
}
