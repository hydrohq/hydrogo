package partners

import (
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
)

type Partner struct {
	sectoral.Base
	LogoURL    string `json:"logo_url"`
	WebsiteURL string `json:"website_url,omitempty"`
	Tier       string `json:"tier,omitempty"`
}

func (v *Partner) Validate() error {
	if v == nil {
		return sectoral.ErrInvalid
	}
	if err := sectoral.ValidateBase(&v.Base); err != nil {
		return err
	}
	if !sectoral.ValidWebURL(v.LogoURL, false) || !sectoral.ValidWebURL(v.WebsiteURL, true) || len(v.Tier) > 80 {
		return sectoral.ErrInvalid
	}
	return nil
}

// Clone creates an unpublished copy with a caller-supplied stable identity.
func (v Partner) Clone(id, slug string) Partner {
	v.Base = sectoral.CloneBase(v.Base, id, slug)
	return v
}
