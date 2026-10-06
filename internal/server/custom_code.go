package server

import (
	"bytes"
	"net/http"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/customcode"
)

const customCodeResponseLimit = 2 << 20

func CustomCodeMiddleware(provider func(*http.Request) (string, string)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || !strings.Contains(r.Header.Get("Accept"), "text/html") || excludedCustomCodePath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			head, body := provider(r)
			if head == "" && body == "" {
				next.ServeHTTP(w, r)
				return
			}
			sources, err := customcode.Validate(head, body)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			capture := &customCodeWriter{header: make(http.Header), limit: customCodeResponseLimit}
			next.ServeHTTP(capture, r)
			if capture.overflow {
				copyHeaders(w.Header(), capture.header)
				w.WriteHeader(capture.status)
				_, _ = w.Write(capture.buf.Bytes())
				_, _ = w.Write(capture.overflowBytes)
				return
			}
			if capture.status != http.StatusOK || !strings.Contains(strings.ToLower(capture.header.Get("Content-Type")), "text/html") {
				copyHeaders(w.Header(), capture.header)
				w.WriteHeader(capture.status)
				_, _ = w.Write(capture.buf.Bytes())
				return
			}
			content := capture.buf.Bytes()
			head = customcode.AddNonces(head)
			body = customcode.AddNonces(body)
			if head != "" {
				content = insertBeforeClosingTag(content, "</head>", head)
			}
			if body != "" {
				content = insertBeforeClosingTag(content, "</body>", body)
			}
			if len(content) > customCodeResponseLimit {
				copyHeaders(w.Header(), capture.header)
				w.WriteHeader(capture.status)
				_, _ = w.Write(capture.buf.Bytes())
				return
			}
			copyHeaders(w.Header(), capture.header)
			if value := customCSPValue(sources); value != "" {
				w.Header().Set("X-Hydro-Custom-CSP", value)
			}
			w.Header().Del("Content-Length")
			w.Header().Del("ETag")
			w.Header().Add("Cache-Control", "private, no-store")
			w.WriteHeader(capture.status)
			_, _ = w.Write(content)
		})
	}
}

func customCSPValue(sources []customcode.Source) string {
	byDirective := make(map[string]map[string]struct{})
	for _, source := range sources {
		if byDirective[source.Directive] == nil {
			byDirective[source.Directive] = make(map[string]struct{})
		}
		byDirective[source.Directive][source.Origin] = struct{}{}
	}
	directives := make([]string, 0, len(byDirective))
	for directive := range byDirective {
		directives = append(directives, directive)
	}
	sort.Strings(directives)
	var result strings.Builder
	for _, directive := range directives {
		origins := make([]string, 0, len(byDirective[directive]))
		for origin := range byDirective[directive] {
			origins = append(origins, origin)
		}
		sort.Strings(origins)
		result.WriteString(directive)
		result.WriteByte(' ')
		result.WriteString(strings.Join(origins, " "))
		result.WriteString("; ")
	}
	return strings.TrimSpace(result.String())
}

func excludedCustomCodePath(path string) bool {
	for _, prefix := range []string{"/admin", "/api", "/auth", "/login", "/media", "/assets", "/static", "/health", "/ready"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func insertBeforeClosingTag(content []byte, tag, snippet string) []byte {
	lower := bytes.ToLower(content)
	if index := bytes.LastIndex(lower, []byte(tag)); index >= 0 {
		result := make([]byte, 0, len(content)+len(snippet))
		result = append(result, content[:index]...)
		result = append(result, snippet...)
		return append(result, content[index:]...)
	}
	return append(content, []byte(snippet)...)
}

type customCodeWriter struct {
	header        http.Header
	buf           bytes.Buffer
	status        int
	limit         int
	overflow      bool
	overflowBytes []byte
}

func (w *customCodeWriter) Header() http.Header { return w.header }
func (w *customCodeWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *customCodeWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.overflow {
		w.overflowBytes = append(w.overflowBytes, data...)
		return len(data), nil
	}
	if w.buf.Len()+len(data) > w.limit {
		w.overflow = true
		w.overflowBytes = append(w.overflowBytes, data...)
		return len(data), nil
	}
	return w.buf.Write(data)
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		dst[key] = append([]string(nil), values...)
	}
}
