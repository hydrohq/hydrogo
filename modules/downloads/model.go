package downloads

import (
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
