package downloads

import (
	"net/url"
	"strings"
	"unicode"

	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
)

type Download struct {
	sectoral.Base
	FileURL       string `json:"file_url"`
	ContentType   string `json:"content_type"`
	FileSize      int64  `json:"file_size"`
	DownloadCount int64  `json:"download_count"`
}

func (v *Download) Validate() error {
	if v == nil {
		return sectoral.ErrInvalid
	}
	if err := sectoral.ValidateBase(&v.Base); err != nil {
		return err
	}
	if !sectoral.ValidWebURL(v.FileURL, false) || len(v.ContentType) > 128 || v.FileSize < 0 || v.DownloadCount < 0 {
		return sectoral.ErrInvalid
	}
	return nil
}

// Clone creates an unpublished copy with a caller-supplied stable identity.
func (v Download) Clone(id, slug string) Download {
	v.Base = sectoral.CloneBase(v.Base, id, slug)
	return v
}

// PDFCoverURL returns the derivative route only for a same-origin media asset
// URL with a safe, single-segment ID.
func PDFCoverURL(fileURL, contentType string) string {
	if !strings.EqualFold(strings.TrimSpace(contentType), "application/pdf") {
		return ""
	}
	parsed, err := url.Parse(fileURL)
	if err != nil || parsed.IsAbs() || parsed.RawQuery != "" || parsed.Fragment != "" || !strings.HasPrefix(parsed.Path, "/media/") {
		return ""
	}
	id, err := url.PathUnescape(strings.TrimPrefix(parsed.EscapedPath(), "/media/"))
	if err != nil || id == "" || len(id) > 128 {
		return ""
	}
	for _, r := range id {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_') {
			return ""
		}
	}
	return "/media/" + url.PathEscape(id) + "?variant=pdf-cover"
}
