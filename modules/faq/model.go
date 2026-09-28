package faq

import (
	"github.com/ardaozyildirim/hydrogo/internal/sectoral"
	"strings"
)

type Question struct {
	sectoral.Base
	Category     string `json:"category"`
	QuestionText string `json:"question"`
	Answer       string `json:"answer"`
}

func (v *Question) Validate() error {
	if v == nil {
		return sectoral.ErrInvalid
	}
	if err := sectoral.ValidateBase(&v.Base); err != nil {
		return err
	}
	if strings.TrimSpace(v.Category) == "" || len(v.Category) > 128 || strings.TrimSpace(v.QuestionText) == "" || len(v.QuestionText) > 1000 || strings.TrimSpace(v.Answer) == "" || len(v.Answer) > 10000 {
		return sectoral.ErrInvalid
	}
	return nil
}

// Clone creates an unpublished copy with a caller-supplied stable identity.
func (v Question) Clone(id, slug string) Question {
	v.Base = sectoral.CloneBase(v.Base, id, slug)
	return v
}
