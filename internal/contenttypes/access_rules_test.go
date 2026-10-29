package contenttypes

import (
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

func TestEvaluateRule_Keywords(t *testing.T) {
	anon := Principal{Anonymous: true}
	member := Principal{ID: "mem_1", Email: "m@x.com", Role: "member", IsMember: true}
	admin := Principal{ID: "adm_1", Email: "a@x.com", Role: "admin", IsAdmin: true}

	cases := []struct {
		rule         string
		principal    Principal
		isReadOrList bool
		want         bool
	}{
		{"public", anon, false, true},
		{"public", member, false, true},
		{"true", anon, false, true},
		{"false", anon, false, false},
		{"false", member, false, false},
		{"false", admin, false, true}, // Admin bypasses everything
		{"admin", anon, false, false},
		{"admin", member, false, false},
		{"admin", admin, false, true},
		{"member", anon, false, false},
		{"member", member, false, true},
		{"member", admin, false, true},
		{"", anon, true, true},   // Empty for read/list is allowed
		{"", anon, false, false}, // Empty for mutation is denied
		{"", member, false, false},
		{"", admin, false, true},
	}

	for _, tc := range cases {
		got, err := EvaluateRule(tc.rule, tc.principal, nil, tc.isReadOrList)
		if err != nil {
			t.Errorf("rule %q error: %v", tc.rule, err)
		}
		if got != tc.want {
			t.Errorf("rule %q (principal %+v, isRead %v) = %v; want %v", tc.rule, tc.principal, tc.isReadOrList, got, tc.want)
		}
	}
}

func TestEvaluateRule_AuthExpressions(t *testing.T) {
	anon := Principal{Anonymous: true}
	member1 := Principal{ID: "u1", Email: "u1@test.com", Role: "member", IsMember: true}
	member2 := Principal{ID: "u2", Email: "u2@test.com", Role: "member", IsMember: true}

	rec1 := map[string]any{"id": "rec_1", "author_id": "u1", "status": "draft"}
	rec2 := map[string]any{"id": "rec_2", "author_id": "u2", "status": "published"}

	// Rule 1: Authenticated member only
	ruleAuth := "@auth.id != ''"
	if got, _ := EvaluateRule(ruleAuth, anon, nil, false); got {
		t.Error("anon should not pass @auth.id != ''")
	}
	if got, _ := EvaluateRule(ruleAuth, member1, nil, false); !got {
		t.Error("member1 should pass @auth.id != ''")
	}

	// Rule 2: Owner check (@auth.id == record.author_id)
	ruleOwner := "@auth.id == record.author_id"
	if got, _ := EvaluateRule(ruleOwner, member1, rec1, false); !got {
		t.Error("member1 should own rec1")
	}
	if got, _ := EvaluateRule(ruleOwner, member2, rec1, false); got {
		t.Error("member2 should NOT own rec1")
	}

	// Rule 3: Published OR Owner
	ruleFeed := "status == 'published' || @auth.id == author_id"
	if got, _ := EvaluateRule(ruleFeed, member1, rec1, true); !got {
		t.Error("member1 should see draft rec1 because they are author")
	}
	if got, _ := EvaluateRule(ruleFeed, member2, rec1, true); got {
		t.Error("member2 should NOT see draft rec1")
	}
	if got, _ := EvaluateRule(ruleFeed, anon, rec2, true); !got {
		t.Error("anon should see published rec2")
	}
}

func TestValidateAccessRule(t *testing.T) {
	schema := collections.Schema{
		ID: "tweets",
		Fields: []collections.Field{
			{ID: "text", Kind: collections.Text},
			{ID: "author_id", Kind: collections.Text},
			{ID: "likes", Kind: collections.Number},
		},
	}

	valid := []string{
		"",
		"public",
		"admin",
		"member",
		"@auth.id != ''",
		"@auth.id == record.author_id",
		"@auth.role == 'admin'",
		"likes > 0",
		"empty(author_id) || @auth.id == author_id",
	}

	for _, rule := range valid {
		if err := ValidateAccessRule(rule, schema); err != nil {
			t.Errorf("expected valid rule %q, got error: %v", rule, err)
		}
	}

	invalid := []string{
		"1 +",
		"@auth.id == non_existent_column",
		"text == 'foo' && unknown_field > 0",
	}

	for _, rule := range invalid {
		if err := ValidateAccessRule(rule, schema); err == nil {
			t.Errorf("expected error for invalid rule %q, got nil", rule)
		}
	}
}

func TestDefinitionValidate_AccessRules(t *testing.T) {
	d := Definition{
		ID:         "messages",
		Label:      "Messages",
		TitleField: "content",
		Fields: []Field{
			{ID: "content", Label: "Content", Kind: "text", Required: true},
			{ID: "sender_id", Label: "Sender", Kind: "text", Required: true},
		},
		Realtime: true,
		AccessRules: &AccessRules{
			List:   "public",
			Read:   "public",
			Create: "@auth.id != '' && @auth.id == record.sender_id",
			Update: "@auth.id == record.sender_id",
			Delete: "@auth.id == record.sender_id",
		},
	}

	if err := d.Validate(); err != nil {
		t.Fatalf("valid definition rejected: %v", err)
	}

	// Invalid rule should fail validation
	d.AccessRules.Create = "@auth.id == bad_field"
	if err := d.Validate(); err == nil {
		t.Fatal("definition with bad field in access rule should fail validation")
	}
}
