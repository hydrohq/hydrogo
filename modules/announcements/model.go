package announcements

import (
	"github.com/ardaozyildirim/hydrogo/internal/forms"
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"time"
)

type Announcement struct {
	sectoral.Base
	LinkURL     string    `json:"link_url,omitempty"`
	StartsAt    time.Time `json:"starts_at,omitempty"`
	EndsAt      time.Time `json:"ends_at,omitempty"`
	Dismissible bool      `json:"dismissible"`
}

// ExitIntentProps configures the optional desktop abandonment prompt. The
// embedded form retains the Forms module's CSRF, honeypot and submission lock.
type ExitIntentProps struct {
	Form             forms.Form
	Action           string
	CSRFToken        string
	SubmissionToken  string
	TurnstileSiteKey string
	Headline         string
	Description      string
	SubmitLabel      string
}

func exitIntentText(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (v *Announcement) Validate() error {
	if v == nil {
		return sectoral.ErrInvalid
	}
	if err := sectoral.ValidateBase(&v.Base); err != nil {
		return err
	}
	if len(v.Body) == 0 || !sectoral.ValidWebURL(v.LinkURL, true) || (!v.StartsAt.IsZero() && !v.EndsAt.IsZero() && !v.StartsAt.Before(v.EndsAt)) {
		return sectoral.ErrInvalid
	}
	return nil
}

// Clone creates an unpublished copy with a caller-supplied stable identity.
func (v Announcement) Clone(id, slug string) Announcement {
	v.Base = sectoral.CloneBase(v.Base, id, slug)
	return v
}
