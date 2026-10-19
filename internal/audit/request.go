package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
)

const (
	// maxCapturedBody is how much of a request body is read for the audit
	// trail; the rest still reaches the handler untouched.
	maxCapturedBody = 64 << 10
	// maxDetailBytes keeps the stored details under the 4 KiB audit limit.
	maxDetailBytes  = 3800
	maxStringLength = 200
	maxDepth        = 8
	maxTargetLength = 200
	redactedValue   = "[redacted]"
)

// sensitiveKeyParts mark JSON keys whose values are never stored.
var sensitiveKeyParts = []string{
	"password", "passwd", "secret", "token", "authorization", "cookie", "credential",
	"private", "signature", "api_key", "apikey", "totp", "mfa", "recovery", "otp",
}

// DetailRecorder writes an audit event with JSON details.
type DetailRecorder interface {
	RecordDetails(ctx context.Context, action, targetID string, before, after []byte) error
}

// RequestLog records what authenticated users send to the admin API.
type RequestLog struct {
	Recorder DetailRecorder
	// SkipPath reports paths that are never captured (uploads, noisy endpoints).
	SkipPath func(path string) bool
}

// Middleware records one "admin.request" event per state-changing admin API
// request made by a signed-in user, with the redacted JSON body. Bodies are
// read through a bounded buffer and passed on unchanged.
func (l RequestLog) Middleware(next http.Handler) http.Handler {
	if l.Recorder == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !capturable(r) || (l.SkipPath != nil && l.SkipPath(r.URL.Path)) {
			next.ServeHTTP(w, r)
			return
		}
		var captured []byte
		oversize := false
		if r.Body != nil && r.Body != http.NoBody {
			head, _ := io.ReadAll(io.LimitReader(r.Body, maxCapturedBody+1))
			oversize = len(head) > maxCapturedBody
			captured = head
			r.Body = struct {
				io.Reader
				io.Closer
			}{io.MultiReader(bytes.NewReader(head), r.Body), r.Body}
		}
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)

		user := auth.UserFromContext(r.Context())
		if user == nil || !user.Active {
			return
		}
		details := RequestDetails(r.Method, r.URL.Path, r.URL.RawQuery, recorder.status, r.RemoteAddr, r.UserAgent(), captured, oversize)
		target := r.Method + " " + r.URL.Path
		if len(target) > maxTargetLength {
			target = target[:maxTargetLength]
		}
		_ = l.Recorder.RecordDetails(context.WithoutCancel(r.Context()), "admin.request", target, nil, details)
	})
}

func capturable(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v1/admin/") {
		return false
	}
	contentType := r.Header.Get("Content-Type")
	return contentType == "" || strings.HasPrefix(contentType, "application/json")
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (s *statusRecorder) WriteHeader(status int) {
	if !s.wroteHeader {
		s.status, s.wroteHeader = status, true
	}
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Write(p []byte) (int, error) {
	s.wroteHeader = true
	return s.ResponseWriter.Write(p)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// RequestDetails builds the JSON stored for one request. The result is always
// valid JSON under the audit size limit.
func RequestDetails(method, path, query string, status int, ip, userAgent string, body []byte, oversize bool) []byte {
	details := map[string]any{"method": method, "path": path, "status": status, "ip": ip, "user_agent": truncate(userAgent, 160)}
	if query != "" {
		details["query"] = redactQuery(query)
	}
	switch {
	case oversize:
		details["body"] = "[body larger than 64 KiB not captured]"
	case len(bytes.TrimSpace(body)) > 0:
		details["body"] = RedactJSON(body)
	}
	for limit := maxStringLength; limit >= 20; limit /= 2 {
		out, err := json.Marshal(details)
		if err == nil && len(out) <= maxDetailBytes {
			return out
		}
		if details["body"] != nil {
			details["body"] = shrink(details["body"], limit, 0)
		}
	}
	delete(details, "body")
	details["body_truncated"] = true
	out, _ := json.Marshal(details)
	return out
}

// RedactJSON parses a JSON document and replaces the value of every
// sensitive key. Non-JSON input is reported, not stored.
func RedactJSON(body []byte) any {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return "[not valid JSON]"
	}
	return redact(value, 0)
}

func redact(value any, depth int) any {
	if depth > maxDepth {
		return "[too deep]"
	}
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			if sensitiveKey(key) {
				out[key] = redactedValue
				continue
			}
			out[key] = redact(item, depth+1)
		}
		return out
	case []any:
		out := make([]any, 0, len(v))
		for i, item := range v {
			if i >= 50 {
				out = append(out, "[+more items]")
				break
			}
			out = append(out, redact(item, depth+1))
		}
		return out
	case string:
		return truncate(v, maxStringLength)
	default:
		return v
	}
}

func shrink(value any, limit, depth int) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = shrink(item, limit, depth+1)
		}
		return out
	case []any:
		if len(v) > 10 {
			v = v[:10]
		}
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = shrink(item, limit, depth+1)
		}
		return out
	case string:
		return truncate(v, limit)
	default:
		return v
	}
}

func sensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, part := range sensitiveKeyParts {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}

func redactQuery(raw string) string {
	parts := strings.Split(raw, "&")
	for i, part := range parts {
		key, _, _ := strings.Cut(part, "=")
		if sensitiveKey(key) {
			parts[i] = key + "=" + redactedValue
		}
	}
	return truncate(strings.Join(parts, "&"), 300)
}

func truncate(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit]) + "…(" + itoa(len(runes)) + " chars)"
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
