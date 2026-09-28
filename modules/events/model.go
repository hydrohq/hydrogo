package events

import (
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"time"
)

type Event struct {
	sectoral.Base
	StartsAt        time.Time `json:"starts_at"`
	EndsAt          time.Time `json:"ends_at"`
	Venue           string    `json:"venue,omitempty"`
	Timezone        string    `json:"timezone"`
	RegistrationURL string    `json:"registration_url,omitempty"`
}

func (v *Event) Validate() error {
	if v == nil {
		return sectoral.ErrInvalid
	}
	if err := sectoral.ValidateBase(&v.Base); err != nil {
		return err
	}
	if v.StartsAt.IsZero() || v.EndsAt.IsZero() || !v.StartsAt.Before(v.EndsAt) || len(v.Venue) > 512 || !sectoral.ValidTimezone(v.Timezone) || !sectoral.ValidWebURL(v.RegistrationURL, true) {
		return sectoral.ErrInvalid
	}
	return nil
}

// Clone creates an unpublished copy with a caller-supplied stable identity.
func (v Event) Clone(id, slug string) Event { v.Base = sectoral.CloneBase(v.Base, id, slug); return v }
