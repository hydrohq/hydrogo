package security

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

//go:embed http_error.html
var httpErrorHTML string

var httpErrorTmpl = template.Must(template.New("http_error").Parse(httpErrorHTML))

type httpErrorView struct {
	StatusCode int
	Code       string
	Title      string
	Message    string
	RequestID  string
}

const maxBufferedErrorResponse = 2 << 20

// ErrorResponse is the stable machine-readable shape used by HTTP API errors.
type ErrorResponse struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id"`
}

// ErrorMiddleware normalizes plain-text HTTP errors emitted by handlers while
// preserving status codes, headers, and successful/streaming responses.
func ErrorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capture := newErrorCapture(w)
		next.ServeHTTP(capture, r)
		capture.commit(r)
	})
}

type errorCapture struct {
	destination http.ResponseWriter
	header      http.Header
	body        bytes.Buffer
	status      int
	passthrough bool
}

// newErrorCapture starts from a copy of the outer headers: handlers must see
// what outer middleware already set (notably the Content-Security-Policy they
// merge inline hashes into); with an empty map a merged policy replaced the
// full one and lost default-src/style-src.
func newErrorCapture(destination http.ResponseWriter) *errorCapture {
	return &errorCapture{destination: destination, header: destination.Header().Clone()}
}

func (c *errorCapture) Header() http.Header { return c.header }

// Unwrap lets http.ResponseController reach optional capabilities of the
// underlying writer without exposing the buffered error body directly.
func (c *errorCapture) Unwrap() http.ResponseWriter { return c.destination }

func (c *errorCapture) WriteHeader(status int) {
	if c.status != 0 {
		return
	}
	c.status = status
}

func (c *errorCapture) Write(p []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	if c.passthrough {
		return c.destination.Write(p)
	}
	if c.body.Len()+len(p) > maxBufferedErrorResponse {
		if c.status >= http.StatusBadRequest {
			// Never switch an error response to passthrough: doing so would
			// bypass the status-based redaction performed by commit.
			return len(p), nil
		}
		c.flushPassthrough()
		return c.destination.Write(p)
	}
	return c.body.Write(p)
}

func (c *errorCapture) Flush() {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	if c.status >= http.StatusBadRequest {
		return
	}
	c.flushPassthrough()
	if flusher, ok := c.destination.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (c *errorCapture) flushPassthrough() {
	if c.passthrough {
		return
	}
	c.passthrough = true
	copyHeaders(c.destination.Header(), c.header)
	c.destination.WriteHeader(c.status)
	if c.body.Len() > 0 {
		_, _ = c.destination.Write(c.body.Bytes())
		c.body.Reset()
	}
}

func wantsBrowserHTML(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		return false
	}
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html") {
		return false
	}
	return strings.Contains(accept, "text/html")
}

func (c *errorCapture) commit(r *http.Request) {
	if c.passthrough {
		return
	}
	if c.status == 0 {
		c.status = http.StatusOK
	}

	if c.status >= http.StatusBadRequest && wantsBrowserHTML(r) {
		msg := strings.TrimSpace(c.body.String())
		if c.status >= http.StatusInternalServerError {
			msg = publicServerErrorMessage(c.status)
		}
		if msg == "" {
			msg = publicServerErrorMessage(c.status)
		}

		title := "Bir Hata Oluştu"
		switch c.status {
		case http.StatusBadRequest:
			title = "Geçersiz İstek"
		case http.StatusUnauthorized:
			title = "Yetkilendirme Gerekli"
		case http.StatusForbidden:
			title = "Erişim Reddedildi"
		case http.StatusNotFound:
			title = "Sayfa Bulunamadı"
		case http.StatusTooManyRequests:
			title = "Çok Fazla İstek"
		case http.StatusInternalServerError:
			title = "Sunucu Hatası"
		case http.StatusBadGateway, http.StatusServiceUnavailable:
			title = "Servis Geçici Olarak Kullanılamıyor"
		}

		c.header.Set("Content-Type", "text/html; charset=utf-8")
		c.header.Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' data: https://fonts.gstatic.com; img-src 'self' data: blob: https:; frame-ancestors 'none';")
		c.header.Del("Content-Length")
		c.body.Reset()
		_ = httpErrorTmpl.Execute(&c.body, httpErrorView{
			StatusCode: c.status,
			Code:       ErrorCode(c.status),
			Title:      title,
			Message:    msg,
			RequestID:  RequestIDFromContext(r.Context()),
		})
	} else if c.status >= http.StatusInternalServerError {
		payload, err := json.Marshal(ErrorResponse{
			Code:      ErrorCode(c.status),
			Message:   publicServerErrorMessage(c.status),
			RequestID: RequestIDFromContext(r.Context()),
		})
		if err == nil {
			c.header.Set("Content-Type", "application/json; charset=utf-8")
			c.header.Del("Content-Length")
			c.body.Reset()
			_, _ = c.body.Write(payload)
			_, _ = c.body.WriteString("\n")
		}
	} else if c.status >= http.StatusBadRequest && !isJSON(c.header.Get("Content-Type")) {
		payload, err := json.Marshal(ErrorResponse{
			Code:      ErrorCode(c.status),
			Message:   strings.TrimSpace(c.body.String()),
			RequestID: RequestIDFromContext(r.Context()),
		})
		if err == nil {
			c.header.Set("Content-Type", "application/json; charset=utf-8")
			c.header.Del("Content-Length")
			c.body.Reset()
			_, _ = c.body.Write(payload)
			_, _ = c.body.WriteString("\n")
		}
	}
	copyHeaders(c.destination.Header(), c.header)
	c.destination.WriteHeader(c.status)
	_, _ = c.destination.Write(c.body.Bytes())
}

// copyHeaders makes destination equal to source, so headers the handler
// deleted from its copy are removed from the real response too.
func copyHeaders(destination, source http.Header) {
	for key := range destination {
		if _, ok := source[key]; !ok {
			delete(destination, key)
		}
	}
	for key, values := range source {
		destination[key] = append([]string(nil), values...)
	}
}

func isJSON(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(contentType), "application/json")
}

func ErrorCode(status int) string {
	known := map[int]string{
		http.StatusBadRequest:            "bad_request",
		http.StatusUnauthorized:          "unauthorized",
		http.StatusForbidden:             "forbidden",
		http.StatusNotFound:              "not_found",
		http.StatusMethodNotAllowed:      "method_not_allowed",
		http.StatusConflict:              "conflict",
		http.StatusRequestEntityTooLarge: "request_entity_too_large",
		http.StatusUnprocessableEntity:   "unprocessable_entity",
		http.StatusTooManyRequests:       "too_many_requests",
		http.StatusInternalServerError:   "internal_server_error",
		http.StatusServiceUnavailable:    "service_unavailable",
	}
	if code, ok := known[status]; ok {
		return code
	}
	return "http_" + strconv.Itoa(status)
}

func publicServerErrorMessage(status int) string {
	if status == http.StatusServiceUnavailable {
		return "service unavailable"
	}
	return "internal server error"
}
