package testimonials

import (
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"strings"
)

type Testimonial struct {
	sectoral.Base
	Author       string `json:"author"`
	Organization string `json:"organization,omitempty"`
	Rating       int    `json:"rating"`
	Quote        string `json:"quote"`
}

func (v *Testimonial) Validate() error {
	if v == nil {
		return sectoral.ErrInvalid
	}
	if err := sectoral.ValidateBase(&v.Base); err != nil {
		return err
	}
	if strings.TrimSpace(v.Author) == "" || len(v.Author) > 256 || len(v.Organization) > 256 || v.Rating < 1 || v.Rating > 5 || strings.TrimSpace(v.Quote) == "" || len(v.Quote) > 10000 {
		return sectoral.ErrInvalid
	}
	return nil
}

// Clone creates an unpublished copy with a caller-supplied stable identity.
func (v Testimonial) Clone(id, slug string) Testimonial {
	v.Base = sectoral.CloneBase(v.Base, id, slug)
	return v
}
