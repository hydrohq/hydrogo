package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

type extensionChannelRegistryFake struct {
	boundID, boundChallenge string
	boundPrincipal          adminstudio.ChannelPrincipal
	revokedID               string
	revokedPrincipal        adminstudio.ChannelPrincipal
	bindErr                 error
}

func (f *extensionChannelRegistryFake) BindChannel(id, challenge string, principal adminstudio.ChannelPrincipal) error {
	f.boundID, f.boundChallenge, f.boundPrincipal = id, challenge, principal
	return f.bindErr
}

func (f *extensionChannelRegistryFake) RevokeChannel(id string, principal adminstudio.ChannelPrincipal) {
	f.revokedID, f.revokedPrincipal = id, principal
}

func TestExtensionChannelMutationRoutesEnforceRoleCSRFAndSessionBinding(t *testing.T) {
	registry := &extensionChannelRegistryFake{}
	routes, err := router.New([]router.Route{
		{Pattern: "POST /admin/extensions/channels/bind", Access: router.Protected, Handler: adminExtensionChannelBindHandler(registry)},
		{Pattern: "POST /admin/extensions/channels/revoke", Access: router.Protected, Handler: adminExtensionChannelRevokeHandler(registry)},
	}, func(*http.Request) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	handler := security.CSRFMiddleware(false, nil)(routes)
	admin := &auth.User{ID: "admin-1", Active: true, Role: auth.RoleAdmin}
	session := &auth.Session{ID: "session-1", UserID: admin.ID}
	request := func(path, body, csrf string, user *auth.User, current *auth.Session) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "valid-token"})
		if csrf != "" {
			req.Header.Set(security.CSRFHeaderName, csrf)
		}
		scope, err := tenant.NewScope("tenant-1", "edition-1")
		if err != nil {
			t.Fatal(err)
		}
		req = req.WithContext(tenant.WithScope(req.Context(), scope))
		req = req.WithContext(auth.WithSession(req.Context(), current))
		req = req.WithContext(auth.WithUser(req.Context(), user))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	bindBody := `{"channel_id":"channel-1","challenge":"challenge-1","tenant_id":"attacker-tenant","user_id":"attacker-user","session_id":"attacker-session"}`
	revokeBody := `{"channel_id":"channel-1"}`
	for _, tc := range []struct {
		name, path, body string
		user             *auth.User
		session          *auth.Session
		csrf             string
	}{
		{name: "missing CSRF", path: "/admin/extensions/channels/bind", body: bindBody, user: admin, session: session},
		{name: "editor", path: "/admin/extensions/channels/bind", body: bindBody, user: &auth.User{ID: "editor", Active: true, Role: auth.RoleEditor}, session: &auth.Session{ID: "editor-session", UserID: "editor"}, csrf: "valid-token"},
		{name: "session user mismatch", path: "/admin/extensions/channels/revoke", body: revokeBody, user: admin, session: &auth.Session{ID: "other-session", UserID: "different-user"}, csrf: "valid-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			beforeBind, beforeRevoke := registry.boundID, registry.revokedID
			response := request(tc.path, tc.body, tc.csrf, tc.user, tc.session)
			if response.Code != http.StatusForbidden || registry.boundID != beforeBind || registry.revokedID != beforeRevoke {
				t.Fatalf("denied request status=%d bound=%q revoked=%q", response.Code, registry.boundID, registry.revokedID)
			}
		})
	}
	if response := request("/admin/extensions/channels/bind", bindBody, "valid-token", admin, session); response.Code != http.StatusNoContent {
		t.Fatalf("Admin bind status=%d body=%s", response.Code, response.Body.String())
	}
	wantPrincipal := adminstudio.ChannelPrincipal{SessionID: session.ID, UserID: admin.ID, TenantID: "tenant-1"}
	if registry.boundID != "channel-1" || registry.boundChallenge != "challenge-1" || registry.boundPrincipal != wantPrincipal {
		t.Fatalf("bind forwarded untrusted or incorrect data: id=%q challenge=%q principal=%+v", registry.boundID, registry.boundChallenge, registry.boundPrincipal)
	}
	if response := request("/admin/extensions/channels/revoke", revokeBody, "valid-token", admin, session); response.Code != http.StatusNoContent {
		t.Fatalf("Admin revoke status=%d body=%s", response.Code, response.Body.String())
	}
	if registry.revokedID != "channel-1" || registry.revokedPrincipal != wantPrincipal {
		t.Fatalf("revoke forwarded incorrect data: id=%q principal=%+v", registry.revokedID, registry.revokedPrincipal)
	}
	if response := request("/admin/extensions/channels/bind", bindBody, "valid-token", admin, session); response.Code != http.StatusNoContent {
		t.Fatalf("Admin channel bind handler status=%d", response.Code)
	}
	registry.bindErr = errors.New("private channel state")
	response := request("/admin/extensions/channels/bind", bindBody, "valid-token", admin, session)
	if response.Code != http.StatusForbidden || strings.Contains(response.Body.String(), "private channel state") {
		t.Fatalf("channel denial leaked registry details: status=%d body=%s", response.Code, response.Body.String())
	}
}
