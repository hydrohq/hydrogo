package team

import (
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"strings"
)

type Member struct {
	sectoral.Base
	Role        string          `json:"role"`
	Biography   string          `json:"biography,omitempty"`
	PhotoURL    string          `json:"photo_url,omitempty"`
	Email       string          `json:"email,omitempty"`
	SocialLinks []sectoral.Link `json:"social_links,omitempty"`
}

func (v *Member) Validate() error {
	if v == nil {
		return sectoral.ErrInvalid
	}
	if err := sectoral.ValidateBase(&v.Base); err != nil {
		return err
	}
	if strings.TrimSpace(v.Role) == "" || len(v.Role) > 128 || len(v.Biography) > 20000 || !sectoral.ValidWebURL(v.PhotoURL, true) || !sectoral.ValidEmail(v.Email) || !sectoral.ValidLinks(v.SocialLinks, 12) {
		return sectoral.ErrInvalid
	}
	return nil
}

// Clone creates an unpublished copy with a caller-supplied stable identity.
func (v Member) Clone(id, slug string) Member {
	v.Base = sectoral.CloneBase(v.Base, id, slug)
	return v
}
