package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestAdminPasswordResetLinkIsOneUseAndResetsTargetAccount(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "admin-reset-link.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	users := authstore.NewUserStore(db, database.SQLite)
	now := time.Now().UTC()
	adminHash, err := auth.HashPassword("AdminPass!234", auth.TestArgon2Params)
	if err != nil {
		t.Fatal(err)
	}
	admin := &auth.User{ID: "reset-admin", Email: "admin@example.test", PasswordHash: adminHash, Role: auth.RoleAdmin, Active: true, CreatedAt: now, UpdatedAt: now}
	targetHash, err := auth.HashPassword("OldPass!2345", auth.TestArgon2Params)
	if err != nil {
		t.Fatal(err)
	}
	target := &auth.User{ID: "reset-target", Email: "person@example.test", PasswordHash: targetHash, Role: auth.RoleEditor, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := users.Create(ctx, admin); err != nil {
		t.Fatal(err)
	}
	if err := users.Create(ctx, target); err != nil {
		t.Fatal(err)
	}
	transactions := database.NewTenantTransactor(db, database.SQLite, false)
	recovery := &auth.PasswordRecovery{Users: users, Tokens: users, Sessions: authstore.NewSessionStore(db, database.SQLite), ArgonParams: auth.TestArgon2Params}
	handler := auth.HTTPHandler{Service: auth.NewAuthService(users, authstore.NewSessionStore(db, database.SQLite), nil, auth.TestArgon2Params), Recovery: recovery, RecoveryDB: db, Transactions: transactions, PublicOrigin: "https://hydro.example.test"}
	var generate http.Handler
	var confirm http.Handler
	for _, route := range handler.Routes() {
		switch route.Pattern {
		case "POST /auth/users/{id}/password-reset-link":
			generate = route.Handler
		case "POST /auth/password-reset/confirm":
			confirm = route.Handler
		}
	}
	if generate == nil || confirm == nil {
		t.Fatal("password reset routes missing")
	}
	request := httptest.NewRequest(http.MethodPost, "/auth/users/reset-target/password-reset-link", nil)
	request.SetPathValue("id", target.ID)
	request = request.WithContext(auth.WithUser(ctx, admin))
	response := httptest.NewRecorder()
	generate.ServeHTTP(response, request)
	var payload struct {
		URL string `json:"url"`
	}
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &payload) != nil {
		t.Fatalf("reset link status=%d response=%s", response.Code, response.Body.String())
	}
	parsed, err := url.Parse(payload.URL)
	if err != nil || parsed.Host != "hydro.example.test" || !strings.HasPrefix(parsed.Path, "/password-reset") || len(parsed.Query().Get("token")) != 64 {
		t.Fatalf("invalid reset URL %q", payload.URL)
	}
	form := url.Values{"token": {parsed.Query().Get("token")}, "password": {"NewSecurePass!6789"}}
	resetRequest := httptest.NewRequest(http.MethodPost, "/auth/password-reset/confirm", strings.NewReader(form.Encode()))
	resetRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resetResponse := httptest.NewRecorder()
	confirm.ServeHTTP(resetResponse, resetRequest)
	loaded, err := users.GetByID(ctx, target.ID)
	valid, verifyErr := auth.VerifyPassword("NewSecurePass!6789", loaded.PasswordHash)
	if resetResponse.Code != http.StatusNoContent || err != nil || verifyErr != nil || !valid {
		t.Fatalf("reset status=%d user=%+v valid=%v err=%v", resetResponse.Code, loaded, valid, err)
	}
	usedAgain := httptest.NewRecorder()
	confirm.ServeHTTP(usedAgain, resetRequest)
	if usedAgain.Code != http.StatusUnauthorized {
		t.Fatalf("reset token reused with status=%d", usedAgain.Code)
	}
}
