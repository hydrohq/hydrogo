package v1

import "testing"

func TestTaxonomyContractValidatesLocalizedRelations(t *testing.T) {
	terms := []TaxonomyTerm{{ID: "news", Slug: "news", Kind: "category", Labels: map[string]string{"en": "News"}}, {ID: "release", Slug: "release", Kind: "tag", Parent: "news", Labels: map[string]string{"tr": "Sürüm"}}}
	if err := ValidateTaxonomy(terms, []TaxonomyRelation{{ContentID: "article-1", TermIDs: []string{"news", "release"}}}); err != nil {
		t.Fatal(err)
	}
}

func TestTaxonomyContractRejectsCyclesAndUnknownRelations(t *testing.T) {
	cycle := []TaxonomyTerm{{ID: "a", Slug: "a", Kind: "category", Parent: "b", Labels: map[string]string{"en": "A"}}, {ID: "b", Slug: "b", Kind: "category", Parent: "a", Labels: map[string]string{"en": "B"}}}
	if err := ValidateTaxonomy(cycle, nil); err == nil {
		t.Fatal("expected cycle rejection")
	}
	terms := []TaxonomyTerm{{ID: "a", Slug: "a", Kind: "category", Labels: map[string]string{"en": "A"}}}
	if err := ValidateTaxonomy(terms, []TaxonomyRelation{{ContentID: "x", TermIDs: []string{"missing"}}}); err == nil {
		t.Fatal("expected unknown relation rejection")
	}
}

func TestStableTaxonomyTermsDoesNotMutateInput(t *testing.T) {
	input := []TaxonomyTerm{{ID: "z"}, {ID: "a"}}
	got := StableTaxonomyTerms(input)
	if got[0].ID != "a" || input[0].ID != "z" {
		t.Fatalf("unexpected sort/input: %#v %#v", got, input)
	}
}
