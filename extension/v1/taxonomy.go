package v1

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
)

const (
	MaxTerms            = 1000
	MaxLabelsPerTerm    = 32
	MaxLabelLength      = 256
	MaxSlugLength       = 128
	MaxRelatedPerEntity = 100
)

var taxonomySlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// TaxonomyTerm is a localized category or tag. Modules own persistence.
type TaxonomyTerm struct {
	ID     string            `json:"id"`
	Slug   string            `json:"slug"`
	Kind   string            `json:"kind"`
	Parent string            `json:"parent,omitempty"`
	Labels map[string]string `json:"labels"`
}

// TaxonomyRelation links content to terms without prescribing a storage schema.
type TaxonomyRelation struct {
	ContentID string   `json:"content_id"`
	TermIDs   []string `json:"term_ids"`
}

// ValidateTaxonomy applies bounded, deterministic input rules at the SDK boundary.
func ValidateTaxonomy(terms []TaxonomyTerm, relations []TaxonomyRelation) error {
	if len(terms) > MaxTerms {
		return fmt.Errorf("taxonomy exceeds %d terms", MaxTerms)
	}
	seen := make(map[string]struct{}, len(terms))
	parents := make(map[string]string, len(terms))
	for _, term := range terms {
		if err := validateTaxonomyTerm(term); err != nil {
			return err
		}
		if _, ok := seen[term.ID]; ok {
			return fmt.Errorf("duplicate taxonomy term %q", term.ID)
		}
		seen[term.ID] = struct{}{}
		parents[term.ID] = term.Parent
	}
	for id, parent := range parents {
		if parent == "" {
			continue
		}
		if _, ok := seen[parent]; !ok {
			return fmt.Errorf("taxonomy term %q references unknown parent", id)
		}
		if taxonomyCycle(parents, id) {
			return fmt.Errorf("taxonomy parent cycle includes %q", id)
		}
	}
	for _, relation := range relations {
		if strings.TrimSpace(relation.ContentID) == "" || len(relation.ContentID) > 128 {
			return errors.New("taxonomy relation has invalid content id")
		}
		if len(relation.TermIDs) > MaxRelatedPerEntity {
			return fmt.Errorf("content %q exceeds %d taxonomy terms", relation.ContentID, MaxRelatedPerEntity)
		}
		termSeen := make(map[string]struct{}, len(relation.TermIDs))
		for _, termID := range relation.TermIDs {
			if _, ok := seen[termID]; !ok {
				return fmt.Errorf("content %q references unknown taxonomy term", relation.ContentID)
			}
			if _, ok := termSeen[termID]; ok {
				return fmt.Errorf("content %q repeats taxonomy term %q", relation.ContentID, termID)
			}
			termSeen[termID] = struct{}{}
		}
	}
	return nil
}

func validateTaxonomyTerm(term TaxonomyTerm) error {
	if strings.TrimSpace(term.ID) == "" || len(term.ID) > 128 || strings.ContainsAny(term.ID, `/\\`) {
		return fmt.Errorf("taxonomy term has invalid id %q", term.ID)
	}
	if term.Kind != "category" && term.Kind != "tag" {
		return fmt.Errorf("taxonomy term %q has invalid kind", term.ID)
	}
	if len(term.Slug) == 0 || len(term.Slug) > MaxSlugLength || !taxonomySlugPattern.MatchString(term.Slug) {
		return fmt.Errorf("taxonomy term %q has invalid slug", term.ID)
	}
	if len(term.Labels) == 0 || len(term.Labels) > MaxLabelsPerTerm {
		return fmt.Errorf("taxonomy term %q has invalid labels", term.ID)
	}
	for locale, label := range term.Labels {
		if canonical.Locale(locale) == "" || len(label) == 0 || len(label) > MaxLabelLength {
			return fmt.Errorf("taxonomy term %q has invalid label", term.ID)
		}
	}
	return nil
}

func taxonomyCycle(parents map[string]string, start string) bool {
	seen := map[string]struct{}{}
	for current := start; current != ""; current = parents[current] {
		if _, ok := seen[current]; ok {
			return true
		}
		seen[current] = struct{}{}
	}
	return false
}

// StableTaxonomyTerms returns a deterministic copy for manifests and clients.
func StableTaxonomyTerms(terms []TaxonomyTerm) []TaxonomyTerm {
	result := append([]TaxonomyTerm(nil), terms...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
