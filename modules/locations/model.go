package locations

import (
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"strings"
)

type Location struct {
	sectoral.Base
	Address      string            `json:"address"`
	Phone        string            `json:"phone,omitempty"`
	Latitude     float64           `json:"latitude"`
	Longitude    float64           `json:"longitude"`
	MapURL       string            `json:"map_url,omitempty"`
	OpeningHours map[string]string `json:"opening_hours,omitempty"`
}

func (v *Location) Validate() error {
	if v == nil {
		return sectoral.ErrInvalid
	}
	if err := sectoral.ValidateBase(&v.Base); err != nil {
		return err
	}
	if strings.TrimSpace(v.Address) == "" || len(v.Address) > 1000 || !sectoral.ValidWebURL(v.MapURL, true) || v.Latitude < -90 || v.Latitude > 90 || v.Longitude < -180 || v.Longitude > 180 {
		return sectoral.ErrInvalid
	}
	return nil
}

// Clone creates an unpublished copy with a caller-supplied stable identity.
func (v Location) Clone(id, slug string) Location {
	v.Base = sectoral.CloneBase(v.Base, id, slug)
	return v
}
