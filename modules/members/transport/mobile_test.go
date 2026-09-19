package transport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/contracts/customerorders"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/members"
	"github.com/ardaozyildirim/hydrogo/modules/members/transport"
)

type mobileOrderReaderStub struct{}

type memberMobileAuditStub struct{ actions []string }

func (a *memberMobileAuditStub) Record(_ context.Context, action, _ string) error {
	a.actions = append(a.actions, action)
	return nil
}

func (mobileOrderReaderStub) ListOrdersByUserID(context.Context, database.Executor, string, int) ([]customerorders.Order, error) {
	return []customerorders.Order{{ID: "order-1"}}, nil
}

func TestMemberMobileAuthUsesBearerTokensAndRotatesRefresh(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "member-mobile-http.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}

	hash, err := auth.HashPassword("secret1234", auth.DefaultArgon2Params)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, `INSERT INTO members (id,email,password_hash,active,created_at,updated_at) VALUES (?,?,?,?,?,?)`, "member-http", "reader@example.test", hash, 1, now, now); err != nil {
		t.Fatal(err)
	}
	store := newMemoryMemberStore()
	store.members["member-http"] = &members.Member{ID: "member-http", Email: "reader@example.test", PasswordHash: hash, Active: true}
	store.byEmail["reader@example.test"] = store.members["member-http"]
	audit := &memberMobileAuditStub{}
	handler := transport.Handler{
		Service: members.Service{Members: store, Sessions: newMemorySessionStore(), ArgonParams: auth.DefaultArgon2Params},
		Mobile:  &members.MobileAuthService{DB: db, Dialect: database.SQLite},
		Orders:  mobileOrderReaderStub{},
		DB:      db,
		Audit:   audit,
	}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}

	loginBody := `{"email":"reader@example.test","password":"secret1234","device_id":"ios-1","device_name":"Reader phone","platform":"ios"}`
	login := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/member/auth/login", strings.NewReader(loginBody))
	login.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	mux.ServeHTTP(loginResponse, login)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", loginResponse.Code, loginResponse.Body.String())
	}
	if loginResponse.Header().Get("Set-Cookie") != "" {
		t.Fatal("mobile login must not issue a browser cookie")
	}
	var pair auth.TokenPair
	if err := json.NewDecoder(loginResponse.Body).Decode(&pair); err != nil {
		t.Fatal(err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("mobile login returned incomplete token pair")
	}
	if len(audit.actions) != 1 || audit.actions[0] != "member_mobile_auth.login" {
		t.Fatalf("login audit actions = %#v", audit.actions)
	}
	devices := httptest.NewRequest(http.MethodGet, "/api/v1/mobile/member/auth/devices", nil)
	devices.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	devicesResponse := httptest.NewRecorder()
	mux.ServeHTTP(devicesResponse, devices)
	if devicesResponse.Code != http.StatusOK || !strings.Contains(devicesResponse.Body.String(), `"device_id":"ios-1"`) || !strings.Contains(devicesResponse.Body.String(), `"is_current":true`) {
		t.Fatalf("devices status = %d, body = %s", devicesResponse.Code, devicesResponse.Body.String())
	}

	refreshBody, _ := json.Marshal(map[string]string{"refresh_token": pair.RefreshToken})
	refresh := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/member/auth/refresh", bytes.NewReader(refreshBody))
	refresh.Header.Set("Content-Type", "application/json")
	refreshResponse := httptest.NewRecorder()
	mux.ServeHTTP(refreshResponse, refresh)
	if refreshResponse.Code != http.StatusOK {
		t.Fatalf("refresh status = %d, body = %s", refreshResponse.Code, refreshResponse.Body.String())
	}
	var rotated auth.TokenPair
	if err := json.NewDecoder(refreshResponse.Body).Decode(&rotated); err != nil {
		t.Fatal(err)
	}
	if rotated.RefreshToken == pair.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	orders := httptest.NewRequest(http.MethodGet, "/api/v1/mobile/member/orders?limit=1", nil)
	orders.Header.Set("Authorization", "Bearer "+rotated.AccessToken)
	ordersResponse := httptest.NewRecorder()
	mux.ServeHTTP(ordersResponse, orders)
	if ordersResponse.Code != http.StatusOK || !strings.Contains(ordersResponse.Body.String(), `"order-1"`) {
		t.Fatalf("orders status = %d, body = %s", ordersResponse.Code, ordersResponse.Body.String())
	}

	revoke := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/member/auth/revoke", nil)
	revoke.Header.Set("Authorization", "Bearer "+rotated.AccessToken)
	revokeResponse := httptest.NewRecorder()
	mux.ServeHTTP(revokeResponse, revoke)
	if revokeResponse.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d, body = %s", revokeResponse.Code, revokeResponse.Body.String())
	}
	if len(audit.actions) < 3 || audit.actions[len(audit.actions)-1] != "member_mobile_auth.device_revoked" {
		t.Fatalf("revoke audit actions = %#v", audit.actions)
	}
	unauthorized := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/member/auth/revoke-all", nil)
	unauthorized.Header.Set("Authorization", "Bearer "+rotated.AccessToken)
	unauthorizedResponse := httptest.NewRecorder()
	mux.ServeHTTP(unauthorizedResponse, unauthorized)
	if unauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("revoked access status = %d, want %d", unauthorizedResponse.Code, http.StatusUnauthorized)
	}
}
