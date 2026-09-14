package ui

import (
	"errors"
	"net/http"
	"testing"
)

func TestFrontendManifestValidation(t *testing.T) {
	manifest := FrontendManifest{Version: 1,
		Routes:    []FrontendRoute{{Name: "member-login", Method: http.MethodPost, Path: "/member/login", Access: "public"}},
		Selectors: []FrontendSelector{{Name: "member-status", Selector: "[data-member-status]", Purpose: "login feedback"}},
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	manifest.Routes = append(manifest.Routes, FrontendRoute{Name: "member-login", Method: http.MethodGet, Path: "/other", Access: "public"})
	if err := manifest.Validate(); !errors.Is(err, ErrInvalidFrontendManifest) {
		t.Fatalf("duplicate route result = %v", err)
	}
}

func TestFrontendManifestRejectsUnversionedAndUnsafeContracts(t *testing.T) {
	cases := []FrontendManifest{
		{Routes: []FrontendRoute{{Name: "x", Method: http.MethodGet, Path: "/x", Access: "public"}}},
		{Version: 1, Routes: []FrontendRoute{{Name: "x", Method: "TRACE", Path: "/x", Access: "public"}}},
		{Version: 1, Routes: []FrontendRoute{{Name: "x", Method: http.MethodGet, Path: "x", Access: "public"}}},
		{Version: 1, Selectors: []FrontendSelector{{Name: "x", Selector: "[data-x]"}}},
	}
	for i, manifest := range cases {
		if err := manifest.Validate(); !errors.Is(err, ErrInvalidFrontendManifest) {
			t.Errorf("case %d error = %v", i, err)
		}
	}
}
