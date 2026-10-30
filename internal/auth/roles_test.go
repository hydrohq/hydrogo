package auth

import (
	"testing"
)

func TestCanResource(t *testing.T) {
	adminUser := &User{Role: RoleAdmin, Active: true}
	editorUser := &User{Role: RoleEditor, Active: true}
	viewerUser := &User{Role: RoleViewer, Active: true}
	inactiveAdmin := &User{Role: RoleAdmin, Active: false}

	if !CanResource(adminUser, "users", "delete") {
		t.Fatal("expected admin to have full access")
	}
	if CanResource(inactiveAdmin, "users", "read") {
		t.Fatal("expected inactive user to have no access")
	}

	if !CanResource(editorUser, "collection:posts", "create") {
		t.Fatal("expected editor to create posts")
	}
	if CanResource(editorUser, "users", "create") {
		t.Fatal("expected editor to be forbidden from creating users")
	}

	if !CanResource(viewerUser, "collection:posts", "read") {
		t.Fatal("expected viewer to read posts")
	}
	if CanResource(viewerUser, "collection:posts", "create") {
		t.Fatal("expected viewer to be forbidden from creating posts")
	}

	customUser := &User{
		Role:   Role("custom_author"),
		Active: true,
		Permissions: []RolePermission{
			{Resource: "collection:articles", Action: "create"},
			{Resource: "collection:articles", Action: "read"},
			{Resource: "media", Action: "*"},
		},
	}

	if !CanResource(customUser, "collection:articles", "create") {
		t.Fatal("expected custom_author to create articles")
	}
	if CanResource(customUser, "collection:articles", "delete") {
		t.Fatal("expected custom_author to be forbidden from deleting articles")
	}
	if !CanResource(customUser, "media", "delete") {
		t.Fatal("expected custom_author to delete media due to wildcard action")
	}
	if CanResource(customUser, "collection:posts", "read") {
		t.Fatal("expected custom_author to be forbidden from reading posts")
	}
}
