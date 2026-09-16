package taxonomy

import "testing"

func validTerms() []Term {
	return []Term{
		{ID: "news", Slug: "news", Kind: "category", Labels: map[string]string{"en": "News", "tr": "Haberler"}},
		{ID: "release", Slug: "release", Kind: "tag", Parent: "news", Labels: map[string]string{"en": "Release"}},
	}
}

func TestValidateAcceptsLocalizedTermsAndRelations(t *testing.T) {
	err := Validate(validTerms(), []Relation{{ContentID: "article-1", TermIDs: []string{"news", "release"}}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsUnknownParentDuplicateAndCycle(t *testing.T) {
	for name, terms := range map[string][]Term{
		"unknown parent": {{ID: "x", Slug: "x", Kind: "category", Parent: "missing", Labels: map[string]string{"en": "X"}}},
		"duplicate id":   {validTerms()[0], validTerms()[0]},
		"cycle": {
			{ID: "a", Slug: "a", Kind: "category", Parent: "b", Labels: map[string]string{"en": "A"}},
			{ID: "b", Slug: "b", Kind: "category", Parent: "a", Labels: map[string]string{"en": "B"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := Validate(terms, nil); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateRejectsUnsafeAndUnboundedInput(t *testing.T) {
	bad := validTerms()
	bad[0].Slug = "../news"
	if err := Validate(bad, nil); err == nil {
		t.Fatal("expected unsafe slug rejection")
	}
	relation := []Relation{{ContentID: "a", TermIDs: []string{"news", "news"}}}
	if err := Validate(validTerms(), relation); err == nil {
		t.Fatal("expected duplicate relation rejection")
	}
}

func TestStableTerms(t *testing.T) {
	got := StableTerms([]Term{{ID: "z"}, {ID: "a"}})
	if got[0].ID != "a" || got[1].ID != "z" {
		t.Fatalf("unexpected order: %#v", got)
	}
}
