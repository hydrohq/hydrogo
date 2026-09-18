package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/security"
)

type mobileAuditRecorder struct {
	actions []string
	targets []string
}

func (r *mobileAuditRecorder) Record(_ context.Context, action, target string) error {
	r.actions = append(r.actions, action)
	r.targets = append(r.targets, target)
	return nil
}

func TestMobileHTTPLoginAndAuthenticatedDeviceList(t *testing.T) {
	db, dialect := setupTestDB(t)
	users := authstore.NewUserStore(db, dialect)
	sessions := authstore.NewSessionStore(db, dialect)
	mobile := auth.NewMobileAuthService(db, dialect)
	authService := auth.NewAuthService(users, sessions, mobile, auth.DefaultArgon2Params)
	if _, err := authService.CreateUser(context.Background(), "app@example.test", "correct horse battery staple", auth.RoleEditor); err != nil {
		t.Fatalf("create user: %v", err)
	}
	handler := auth.MobileHTTPHandler{Auth: authService, Mobile: mobile}

	login := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/login", bytes.NewBufferString(`{"email":"app@example.test","password":"correct horse battery staple","device_id":"ios-1","device_name":"iPhone","platform":"iOS"}`))
	login.Header.Set("Content-Type", "application/json")
	loginResult := httptest.NewRecorder()
	handler.RouterRoutes()[0].Handler.ServeHTTP(loginResult, login)
	if loginResult.Code != http.StatusOK {
		t.Fatalf("login status = %d, body=%s", loginResult.Code, loginResult.Body.String())
	}
	var pair auth.TokenPair
	if err := json.Unmarshal(loginResult.Body.Bytes(), &pair); err != nil || pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("invalid token response: pair=%+v err=%v", pair, err)
	}

	devices := httptest.NewRequest(http.MethodGet, "/api/v1/mobile/auth/devices", nil)
	devices.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	devicesResult := httptest.NewRecorder()
	handler.RouterRoutes()[2].Handler.ServeHTTP(devicesResult, devices)
	if devicesResult.Code != http.StatusOK {
		t.Fatalf("devices status = %d, body=%s", devicesResult.Code, devicesResult.Body.String())
	}
}

func TestMobileHTTPRejectsCookieOnlyMutation(t *testing.T) {
	db, dialect := setupTestDB(t)
	users := authstore.NewUserStore(db, dialect)
	sessions := authstore.NewSessionStore(db, dialect)
	mobile := auth.NewMobileAuthService(db, dialect)
	authService := auth.NewAuthService(users, sessions, mobile, auth.DefaultArgon2Params)
	user, err := authService.CreateUser(context.Background(), "cookie@example.test", "correct horse battery staple", auth.RoleEditor)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	pair, err := mobile.IssueTokenPair(context.Background(), user.ID, "ios-cookie", "iPhone", "iOS")
	if err != nil {
		t.Fatalf("issue token pair: %v", err)
	}
	handler := auth.MobileHTTPHandler{Auth: authService, Mobile: mobile}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/revoke-all", nil)
	req.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "browser-session"})
	result := httptest.NewRecorder()
	handler.RouterRoutes()[4].Handler.ServeHTTP(result, req)
	if result.Code != http.StatusUnauthorized {
		t.Fatalf("cookie-only mobile request status = %d, want %d", result.Code, http.StatusUnauthorized)
	}
	if _, _, err := mobile.ValidateAccessToken(context.Background(), pair.AccessToken); err != nil {
		t.Fatalf("cookie-only request revoked the bearer session: %v", err)
	}
}

func TestAuthServiceDisablingUserRevokesMobileTokens(t *testing.T) {
	db, dialect := setupTestDB(t)
	users := authstore.NewUserStore(db, dialect)
	sessions := authstore.NewSessionStore(db, dialect)
	mobile := auth.NewMobileAuthService(db, dialect)
	service := auth.NewAuthService(users, sessions, mobile, auth.DefaultArgon2Params)
	user, err := service.CreateUser(context.Background(), "disable-mobile@example.test", "correct horse battery staple", auth.RoleEditor)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	pair, err := mobile.IssueTokenPair(context.Background(), user.ID, "disable-mobile", "iPhone", "iOS")
	if err != nil {
		t.Fatalf("issue token pair: %v", err)
	}
	if err := service.SetUserActive(context.Background(), user.ID, false); err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if _, _, err := mobile.ValidateAccessToken(context.Background(), pair.AccessToken); err != auth.ErrDeviceRevoked {
		t.Fatalf("disabled user's token validation error = %v, want ErrDeviceRevoked", err)
	}
}

func TestMobileHTTPRateLimitsFailuresAndAuditsWithoutEmailDisclosure(t *testing.T) {
	db, dialect := setupTestDB(t)
	users := authstore.NewUserStore(db, dialect)
	sessions := authstore.NewSessionStore(db, dialect)
	mobile := auth.NewMobileAuthService(db, dialect)
	service := auth.NewAuthService(users, sessions, mobile, auth.DefaultArgon2Params)
	if _, err := service.CreateUser(context.Background(), "rate-limit@example.test", "correct horse battery staple", auth.RoleEditor); err != nil {
		t.Fatalf("create user: %v", err)
	}
	audit := &mobileAuditRecorder{}
	handler := auth.MobileHTTPHandler{
		Auth:         service,
		Mobile:       mobile,
		LoginLimiter: security.NewAttemptLimiter(16, 1, time.Minute),
		Audit:        audit,
	}
	request := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/login", strings.NewReader(`{"email":"rate-limit@example.test","password":"wrong-password","device_id":"ios-rate-limit","device_name":"iPhone","platform":"iOS"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "198.51.100.20:12345"
		res := httptest.NewRecorder()
		handler.RouterRoutes()[0].Handler.ServeHTTP(res, req)
		return res
	}
	if res := request(); res.Code != http.StatusUnauthorized {
		t.Fatalf("first invalid login status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
	if res := request(); res.Code != http.StatusTooManyRequests {
		t.Fatalf("second invalid login status = %d, want %d", res.Code, http.StatusTooManyRequests)
	}
	if len(audit.actions) != 1 || audit.actions[0] != "mobile_auth.login_failed" {
		t.Fatalf("audit actions = %v, want one failed login event", audit.actions)
	}
	if len(audit.targets) != 1 || len(audit.targets[0]) != 64 || strings.Contains(audit.targets[0], "rate-limit@example.test") {
		t.Fatalf("failed-login audit target leaked or was not a digest: %q", audit.targets)
	}
}

func TestMobileHTTPDeviceRevocationIsAudited(t *testing.T) {
	db, dialect := setupTestDB(t)
	users := authstore.NewUserStore(db, dialect)
	sessions := authstore.NewSessionStore(db, dialect)
	mobile := auth.NewMobileAuthService(db, dialect)
	service := auth.NewAuthService(users, sessions, mobile, auth.DefaultArgon2Params)
	user, err := service.CreateUser(context.Background(), "audit@example.test", "correct horse battery staple", auth.RoleEditor)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	audit := &mobileAuditRecorder{}
	handler := auth.MobileHTTPHandler{Auth: service, Mobile: mobile, Audit: audit}
	pair, err := mobile.IssueTokenPair(context.Background(), user.ID, "audit-device", "iPhone", "iOS")
	if err != nil {
		t.Fatalf("issue token pair: %v", err)
	}
	revoke := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/auth/revoke", strings.NewReader(`{"device_id":"audit-device"}`))
	revoke.Header.Set("Content-Type", "application/json")
	revoke.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	revokeResult := httptest.NewRecorder()
	handler.RouterRoutes()[3].Handler.ServeHTTP(revokeResult, revoke)
	if revokeResult.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d, body=%s", revokeResult.Code, revokeResult.Body.String())
	}
	if len(audit.actions) != 1 || audit.actions[0] != "mobile_auth.device_revoked" || audit.targets[0] != user.ID {
		t.Fatalf("revocation audit = actions=%v targets=%v", audit.actions, audit.targets)
	}
}
