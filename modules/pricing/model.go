package pricing

import (
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"strings"
)

type Plan struct {
	sectoral.Base
	PriceCents int64    `json:"price_cents"`
	Currency   string   `json:"currency"`
	Interval   string   `json:"interval"`
	Features   []string `json:"features,omitempty"`
	CTAURL     string   `json:"cta_url,omitempty"`
}

func (v *Plan) Validate() error {
	if v == nil {
		return sectoral.ErrInvalid
	}
	if err := sectoral.ValidateBase(&v.Base); err != nil {
		return err
	}
	if v.PriceCents < 0 || len(v.Currency) != 3 || (v.Interval != "monthly" && v.Interval != "yearly" && v.Interval != "one_time") || len(v.Features) > 50 || !sectoral.ValidWebURL(v.CTAURL, true) {
		return sectoral.ErrInvalid
	}
	for _, f := range v.Features {
		if strings.TrimSpace(f) == "" || len(f) > 256 {
			return sectoral.ErrInvalid
		}
	}
	return nil
}

// Clone creates an unpublished copy with a caller-supplied stable identity.
func (v Plan) Clone(id, slug string) Plan { v.Base = sectoral.CloneBase(v.Base, id, slug); return v }
