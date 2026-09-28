package media

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

var ErrInvalidDocument = errors.New("invalid document upload")

// ValidateDocument accepts bounded PDF, legacy DOC, and DOCX payloads by
// extension and file signature. It does not render or execute document content.
func ValidateDocument(filename string, data []byte, maxBytes int64) error {
	if maxBytes <= 0 || int64(len(data)) == 0 || int64(len(data)) > maxBytes {
		return ErrInvalidDocument
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".pdf":
		if !bytes.HasPrefix(bytes.TrimSpace(data), []byte("%PDF-")) {
			return ErrInvalidDocument
		}
	case ".doc":
		if len(data) < 8 || !bytes.Equal(data[:8], []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}) {
			return ErrInvalidDocument
		}
	case ".docx":
		if len(data) < 4 || !bytes.Equal(data[:4], []byte{'P', 'K', 3, 4}) {
			return ErrInvalidDocument
		}
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return ErrInvalidDocument
		}
		found := false
		for _, f := range zr.File {
			if f.Name == "word/document.xml" {
				found = true
			}
			if strings.HasPrefix(f.Name, "/") || strings.Contains(f.Name, "..") {
				return ErrInvalidDocument
			}
		}
		if !found {
			return ErrInvalidDocument
		}
	default:
		return ErrInvalidDocument
	}
	return nil
}

// HotlinkGuard allows direct visits and requests originating from the canonical
// or explicitly allowed hosts. The Referer is parsed as a URL, never string-matched.
func HotlinkGuard(canonicalOrigin string, allowedHosts []string, next http.Handler) http.Handler {
	canonical, _ := url.Parse(canonicalOrigin)
	allowed := map[string]bool{}
	if canonical != nil && canonical.Hostname() != "" {
		allowed[strings.ToLower(canonical.Hostname())] = true
	}
	for _, host := range []string{"google.com", "www.google.com", "bing.com", "www.bing.com", "search.yahoo.com", "duckduckgo.com"} {
		allowed[host] = true
	}
	for _, host := range allowedHosts {
		host = strings.ToLower(strings.TrimSpace(host))
		if host != "" {
			allowed[host] = true
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ref := strings.TrimSpace(r.Referer())
		if ref != "" {
			u, err := url.Parse(ref)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || !allowedRefererHost(strings.ToLower(u.Hostname()), allowed) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func allowedRefererHost(host string, allowed map[string]bool) bool {
	return allowed[host]
}

// ContentTypeForDocument returns a stable type after signature validation.
func ContentTypeForDocument(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".doc":
		return "application/msword"
	}
	return mime.TypeByExtension(filepath.Ext(name))
}

// ReadDocument is useful at bounded upload boundaries where multipart readers
// must not allocate beyond the configured request limit.
func ReadDocument(r io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, ErrInvalidDocument
	}
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, ErrInvalidDocument
	}
	return data, nil
}
