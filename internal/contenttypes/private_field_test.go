package contenttypes

import (
	"net/url"
	"testing"

	"github.com/hydrohq/hydrogo/internal/collections"
)

func TestPrivateFieldsNeverReachThePublicAPI(t *testing.T) {
	definition := Definition{ID: "notes", Public: true, Fields: []Field{
		{ID: "body", Label: "Body", Kind: "text"},
		{ID: "email", Label: "Email", Kind: "text", Private: true},
	}}
	api := PublicAPI{Types: []Definition{definition}}
	got := api.present("notes", "", collections.Record{ID: "n1", Data: map[string]any{"body": "hi", "email": "a@b.c"}})
	if _, leaked := got["email"]; leaked || got["body"] != "hi" {
		t.Fatalf("private field leaked or body lost: %v", got)
	}
	if _, err := parsePublicQuery(url.Values{"sort": {"email"}}, definition); err == nil {
		t.Fatal("sorting by a private field must be rejected")
	}
	bad := Definition{ID: "notes", Fields: []Field{{ID: "email", Label: "E", Kind: "text", Private: true, Filterable: true}}}
	if err := bad.Validate(); err == nil {
		t.Fatal("private + filterable must be rejected")
	}
}

func TestMediaWidgetIsOnlyForTextFields(t *testing.T) {
	ok := Definition{ID: "posts", Label: "Posts", Fields: []Field{{ID: "cover", Label: "Cover", Kind: "text", Widget: "media"}}}
	if err := ok.Validate(); err != nil {
		t.Fatalf("media widget on a text field must be valid: %v", err)
	}
	bad := Definition{ID: "posts", Label: "Posts", Fields: []Field{{ID: "n", Label: "N", Kind: "number", Widget: "media"}}}
	if err := bad.Validate(); err == nil {
		t.Fatal("media widget on a number field must be rejected")
	}
	admin := ok.AdminCollection()
	if admin.Fields[0].Widget != "media" {
		t.Fatalf("admin field widget = %q, want media", admin.Fields[0].Widget)
	}
}
