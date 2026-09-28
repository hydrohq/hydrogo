package certificates

import (
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"strings"
	"time"
)

type Certificate struct {
	sectoral.Base
	Issuer            string    `json:"issuer"`
	CertificateNumber string    `json:"certificate_number,omitempty"`
	IssuedAt          time.Time `json:"issued_at,omitempty"`
	ExpiresAt         time.Time `json:"expires_at,omitempty"`
	DocumentURL       string    `json:"document_url"`
}

func (v *Certificate) Validate() error {
	if v == nil {
		return sectoral.ErrInvalid
	}
	if err := sectoral.ValidateBase(&v.Base); err != nil {
		return err
	}
	if strings.TrimSpace(v.Issuer) == "" || len(v.Issuer) > 256 || len(v.CertificateNumber) > 128 || !sectoral.ValidWebURL(v.DocumentURL, false) || (!v.IssuedAt.IsZero() && !v.ExpiresAt.IsZero() && v.ExpiresAt.Before(v.IssuedAt)) {
		return sectoral.ErrInvalid
	}
	return nil
}

// Clone creates an unpublished copy with a caller-supplied stable identity.
func (v Certificate) Clone(id, slug string) Certificate {
	v.Base = sectoral.CloneBase(v.Base, id, slug)
	return v
}
