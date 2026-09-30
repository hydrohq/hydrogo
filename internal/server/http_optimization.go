package server

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
)

const (
	compressionThreshold = 1024
	responseBufferLimit  = 2 << 20
)

var responseBuffers = sync.Pool{New: func() any { return new(bytes.Buffer) }}

// ETagMiddleware adds a weak entity tag to bounded, anonymous GET responses and
// answers matching conditional requests without writing the response body.
func ETagMiddleware(next http.Handler) http.Handler {
	if next == nil {
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			next.ServeHTTP(w, r)
			return
		}
		capture := newBoundedResponseWriter(w, responseBufferLimit)
		next.ServeHTTP(capture, r)
		if !capture.buffered || capture.status != http.StatusOK || !cacheableResponse(capture.Header()) {
			capture.commitPlain()
			capture.release()
			return
		}
		body := capture.buf.Bytes()
		hash := sha256.Sum256(body)
		etag := `W/"` + hex.EncodeToString(hash[:]) + `"`
		header := capture.Header()
		header.Set("ETag", etag)
		if !headerHasETag(r.Header.Get("If-None-Match"), etag) {
			if modified := header.Get("Last-Modified"); modified != "" && r.Header.Get("If-None-Match") == "" {
				if t, err := http.ParseTime(modified); err == nil {
					if since, parseErr := http.ParseTime(r.Header.Get("If-Modified-Since")); parseErr == nil && !t.Truncate(time.Second).After(since) {
						header.Del("Content-Type")
						header.Del("Content-Length")
						w.WriteHeader(http.StatusNotModified)
						capture.release()
						return
					}
				}
			}
			capture.commitPlain()
			capture.release()
			return
		}
		header.Del("Content-Type")
		header.Del("Content-Length")
		w.WriteHeader(http.StatusNotModified)
		capture.release()
	})
}

func headerHasETag(value, etag string) bool {
	for _, candidate := range strings.Split(value, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag || strings.TrimPrefix(candidate, "W/") == strings.TrimPrefix(etag, "W/") {
			return true
		}
	}
	return false
}

func cacheableResponse(header http.Header) bool {
	if len(header.Values("Set-Cookie")) != 0 {
		return false
	}
	cacheControl := strings.ToLower(header.Get("Cache-Control"))
	csp := strings.ToLower(header.Get("Content-Security-Policy"))
	hasDynamicHTMLNonce := strings.Contains(strings.ToLower(header.Get("Content-Type")), "text/html") && strings.Contains(csp, "nonce-")
	return !strings.Contains(cacheControl, "no-store") && !strings.Contains(cacheControl, "private") && !strings.Contains(cacheControl, "no-cache") && !hasDynamicHTMLNonce
}

// CompressionMiddleware negotiates Brotli and gzip for compressible response
// types. Small and already-compressed responses pass through unchanged.
func CompressionMiddleware(next http.Handler) http.Handler {
	if next == nil {
		return nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoding := negotiatedEncoding(r.Header.Get("Accept-Encoding"))
		if encoding == "" || r.Method == http.MethodHead || r.Header.Get("Range") != "" {
			next.ServeHTTP(w, r)
			return
		}
		capture := newBoundedResponseWriter(w, responseBufferLimit)
		next.ServeHTTP(capture, r)
		if !capture.buffered || capture.status < 200 || capture.status == http.StatusNoContent || capture.status == http.StatusNotModified || capture.buf.Len() < compressionThreshold || !compressible(capture.Header().Get("Content-Type")) || hasContentEncoding(capture.Header()) || strings.Contains(strings.ToLower(capture.Header().Get("Cache-Control")), "no-transform") {
			capture.commitPlain()
			capture.release()
			return
		}
		compressed := responseBuffers.Get().(*bytes.Buffer)
		compressed.Reset()
		err := compressBody(encoding, capture.buf.Bytes(), compressed)
		if err != nil {
			compressed.Reset()
			responseBuffers.Put(compressed)
			capture.commitPlain()
			capture.release()
			return
		}
		header := capture.Header()
		addVary(header, "Accept-Encoding")
		header.Set("Content-Encoding", encoding)
		header.Set("Content-Length", strconv.Itoa(compressed.Len()))
		capture.writer.WriteHeader(capture.status)
		_, _ = w.Write(compressed.Bytes())
		compressed.Reset()
		responseBuffers.Put(compressed)
		capture.release()
	})
}

func addVary(header http.Header, value string) {
	for _, existing := range header.Values("Vary") {
		for _, token := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(token), value) || strings.TrimSpace(token) == "*" {
				return
			}
		}
	}
	header.Add("Vary", value)
}

func negotiatedEncoding(header string) string {
	qualities := map[string]float64{}
	wildcard := -1.0
	for _, item := range strings.Split(header, ",") {
		parts := strings.Split(strings.TrimSpace(item), ";")
		name := strings.ToLower(strings.TrimSpace(parts[0]))
		quality := 1.0
		for _, param := range parts[1:] {
			key, value, ok := strings.Cut(strings.TrimSpace(param), "=")
			if ok && strings.EqualFold(strings.TrimSpace(key), "q") {
				parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
				if err != nil || parsed < 0 || parsed > 1 {
					quality = 0
				} else {
					quality = parsed
				}
			}
		}
		if name == "*" {
			wildcard = quality
		} else if name != "" {
			qualities[name] = quality
		}
	}
	br, hasBR := qualities["br"]
	gzipQuality, hasGzip := qualities["gzip"]
	if !hasBR {
		br = wildcard
	}
	if !hasGzip {
		gzipQuality = wildcard
	}
	if br > 0 && br >= gzipQuality {
		return "br"
	}
	if gzipQuality > 0 {
		return "gzip"
	}
	return ""
}

func compressible(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	return strings.HasPrefix(contentType, "text/") || contentType == "application/json" || contentType == "application/javascript" || contentType == "application/xml" || contentType == "image/svg+xml"
}

func hasContentEncoding(header http.Header) bool { return header.Get("Content-Encoding") != "" }

func compressBody(encoding string, body []byte, out *bytes.Buffer) error {
	switch encoding {
	case "br":
		writer := brotli.NewWriterLevel(out, 4)
		if _, err := writer.Write(body); err != nil {
			return err
		}
		if err := writer.Close(); err != nil {
			return err
		}
	case "gzip":
		writer, err := gzip.NewWriterLevel(out, gzip.BestSpeed)
		if err != nil {
			return err
		}
		if _, err := writer.Write(body); err != nil {
			return err
		}
		if err := writer.Close(); err != nil {
			return err
		}
	default:
		return io.ErrUnexpectedEOF
	}
	return nil
}

type boundedResponseWriter struct {
	writer   http.ResponseWriter
	buf      *bytes.Buffer
	limit    int
	status   int
	buffered bool
	wrote    bool
}

func newBoundedResponseWriter(writer http.ResponseWriter, limit int) *boundedResponseWriter {
	buf := responseBuffers.Get().(*bytes.Buffer)
	buf.Reset()
	return &boundedResponseWriter{writer: writer, buf: buf, limit: limit, status: http.StatusOK, buffered: true}
}

func (w *boundedResponseWriter) Header() http.Header         { return w.writer.Header() }
func (w *boundedResponseWriter) Unwrap() http.ResponseWriter { return w.writer }

func (w *boundedResponseWriter) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.wrote, w.status = true, status
	if !w.buffered {
		w.writer.WriteHeader(status)
	}
}

func (w *boundedResponseWriter) Write(body []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	if w.buffered && w.buf.Len()+len(body) <= w.limit {
		return w.buf.Write(body)
	}
	if w.buffered {
		w.buffered = false
		w.writer.WriteHeader(w.status)
		if w.buf.Len() > 0 {
			if _, err := w.writer.Write(w.buf.Bytes()); err != nil {
				return 0, err
			}
		}
		w.buf.Reset()
	}
	return w.writer.Write(body)
}

func (w *boundedResponseWriter) Flush() {
	if w.buffered {
		w.buffered = false
		if !w.wrote {
			w.WriteHeader(http.StatusOK)
		}
		w.writer.WriteHeader(w.status)
		_, _ = w.writer.Write(w.buf.Bytes())
		w.buf.Reset()
	}
	if flusher, ok := w.writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *boundedResponseWriter) commitPlain() {
	if w.buffered {
		w.writer.WriteHeader(w.status)
		if w.buf.Len() > 0 {
			_, _ = w.writer.Write(w.buf.Bytes())
		}
	}
}

func (w *boundedResponseWriter) release() {
	w.buf.Reset()
	responseBuffers.Put(w.buf)
}
