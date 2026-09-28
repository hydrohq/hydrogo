package portfolio

import (
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"strings"
	"time"
)

type Project struct {
	sectoral.Base
	Client      string    `json:"client,omitempty"`
	Industry    string    `json:"industry"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	CoverURL    string    `json:"cover_url,omitempty"`
	Gallery     []string  `json:"gallery,omitempty"`
}

func (v *Project) Validate() error {
	if v == nil {
		return sectoral.ErrInvalid
	}
	if err := sectoral.ValidateBase(&v.Base); err != nil {
		return err
	}
	if strings.TrimSpace(v.Industry) == "" || len(v.Industry) > 128 || len(v.Client) > 256 || !sectoral.ValidWebURL(v.CoverURL, true) || len(v.Gallery) > 50 {
		return sectoral.ErrInvalid
	}
	for _, u := range v.Gallery {
		if !sectoral.ValidWebURL(u, false) {
			return sectoral.ErrInvalid
		}
	}
	return nil
}

// Clone creates an unpublished copy with a caller-supplied stable identity.
func (v Project) Clone(id, slug string) Project {
	v.Base = sectoral.CloneBase(v.Base, id, slug)
	return v
}
