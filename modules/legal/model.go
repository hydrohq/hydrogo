package legal

import (
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"strings"
	"time"
)

type Document struct {
	sectoral.Base
	DocumentType    string    `json:"document_type"`
	DocumentVersion string    `json:"document_version"`
	EffectiveAt     time.Time `json:"effective_at"`
}

func (v *Document) Validate() error {
	if v == nil {
		return sectoral.ErrInvalid
	}
	if err := sectoral.ValidateBase(&v.Base); err != nil {
		return err
	}
	if (v.DocumentType != "privacy" && v.DocumentType != "terms" && v.DocumentType != "cookie" && v.DocumentType != "custom") || strings.TrimSpace(v.DocumentVersion) == "" || len(v.DocumentVersion) > 32 || v.EffectiveAt.IsZero() || strings.TrimSpace(v.Body) == "" || len(v.Body) > 1<<20 {
		return sectoral.ErrInvalid
	}
	return nil
}

// Clone creates an unpublished copy with a caller-supplied stable identity.
func (v Document) Clone(id, slug string) Document {
	v.Base = sectoral.CloneBase(v.Base, id, slug)
	return v
}
