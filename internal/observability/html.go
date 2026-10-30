package observability

import (
	"bytes"
	"net/http"
	"strings"
)

// HTMLCommentMiddleware is a final production-output guard. It buffers the
// bounded HTML responses emitted by HydroGo and removes ordinary and
// conditional HTML comments before bytes reach the client. JSON, CSS, JS and
// binary responses pass through unchanged.
func HTMLCommentMiddleware(next http.Handler) http.Handler {
	if next == nil {
		return http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), "text/event-stream") ||
			strings.HasSuffix(r.URL.Path, "/subscribe") ||
			strings.HasSuffix(r.URL.Path, "/events") {
			next.ServeHTTP(w, r)
			return
		}
		buffer := &htmlBuffer{header: w.Header(), underlying: w}
		next.ServeHTTP(buffer, r)
		body := buffer.body.Bytes()
		contentType := buffer.header.Get("Content-Type")
		isHTML := strings.HasPrefix(strings.ToLower(contentType), "text/html")
		if contentType == "" && len(body) > 0 {
			isHTML = strings.HasPrefix(http.DetectContentType(body), "text/html")
		}
		if isHTML && buffer.header.Get("Content-Encoding") == "" {
			body = stripHTMLComments(body)
			buffer.header.Del("Content-Length")
		}
		if buffer.status == 0 {
			buffer.status = http.StatusOK
		}
		w.WriteHeader(buffer.status)
		_, _ = w.Write(body)
	})
}

func stripHTMLComments(input []byte) []byte {
	if !bytes.Contains(input, []byte("<!--")) {
		return input
	}
	output := make([]byte, 0, len(input))
	for len(input) > 0 {
		start := bytes.Index(input, []byte("<!--"))
		if start < 0 {
			output = append(output, input...)
			break
		}
		output = append(output, input[:start]...)
		input = input[start+len("<!--"):]
		end := bytes.Index(input, []byte("-->"))
		if end < 0 {
			// An unterminated comment is still never allowed to reach a client.
			break
		}
		input = input[end+len("-->"):]
	}
	return output
}

type htmlBuffer struct {
	header     http.Header
	underlying http.ResponseWriter
	body       bytes.Buffer
	status     int
}

func (b *htmlBuffer) Header() http.Header { return b.header }

func (b *htmlBuffer) WriteHeader(status int) {
	if b.status == 0 {
		b.status = status
	}
}

func (b *htmlBuffer) Write(body []byte) (int, error) {
	if b.status == 0 {
		b.status = http.StatusOK
	}
	return b.body.Write(body)
}

func (b *htmlBuffer) Flush() {
	if b.status == 0 {
		b.status = http.StatusOK
	}
}

func (b *htmlBuffer) Unwrap() http.ResponseWriter { return b.underlying }
