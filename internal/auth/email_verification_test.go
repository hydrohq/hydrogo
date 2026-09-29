package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
)

func TestEmailVerificationHandlerOneUseAndOwnerBound(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(ctx, database.SQLite, filepath.Join(t.TempDir(), "email-verification.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.ApplyMigrations(ctx, db, database.SQLite, migrations.FS); err != nil {
		t.Fatal(err)
	}
	users := authstore.NewUserStore(db, database.SQLite)
	now := time.Now().UTC()
	user := &auth.User{ID: "verified-admin", Email: "admin@example.test", PasswordHash: "unused", Role: auth.RoleAdmin, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := users.Create(ctx, user); err != nil {
		t.Fatal(err)
	}
	var sentTo, link string
	handler := auth.EmailVerificationHandler{Service: auth.EmailVerificationService{Users: users, Tokens: users}, Transactions: database.NewTenantTransactor(db, database.SQLite, false), PublicOrigin: "https://hydro.example.test", Notifier: auth.EmailVerificationNotifierFunc(func(_ context.Context, email, verificationLink string) error {
		sentTo, link = email, verificationLink
		return nil
	})}
	request := httptest.NewRequest("POST", "/auth/me/email/verification", nil).WithContext(auth.WithUser(ctx, user))
	recorder := httptest.NewRecorder()
	handler.Routes()[0].Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusSeeOther || sentTo != user.Email || link == "" {
		t.Fatalf("request status=%d email=%q link=%q", recorder.Code, sentTo, link)
	}
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	verify := httptest.NewRequest("GET", "/auth/email/verify?token="+url.QueryEscape(parsed.Query().Get("token")), nil)
	verified := httptest.NewRecorder()
	handler.Routes()[1].Handler.ServeHTTP(verified, verify)
	if verified.Code != http.StatusSeeOther {
		t.Fatalf("verify status=%d body=%s", verified.Code, verified.Body.String())
	}
	loaded, err := users.GetByID(ctx, user.ID)
	if err != nil || loaded.EmailVerifiedAt == nil {
		t.Fatalf("verified user=%+v err=%v", loaded, err)
	}
	again := httptest.NewRecorder()
	handler.Routes()[1].Handler.ServeHTTP(again, verify)
	if again.Code != http.StatusBadRequest {
		t.Fatalf("verification token was reusable: status=%d", again.Code)
	}
	target := &auth.User{ID: "unverified-editor", Email: "editor@example.test", PasswordHash: "unused", Role: auth.RoleEditor, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := users.Create(ctx, target); err != nil {
		t.Fatal(err)
	}
	handler.AdminUsers = users
	var resendRoute http.Handler
	for _, route := range handler.Routes() {
		if route.Pattern == "POST /auth/users/{id}/email/verification/resend" {
			resendRoute = route.Handler
		}
	}
	if resendRoute == nil {
		t.Fatal("admin resend route missing")
	}
	forbiddenRequest := httptest.NewRequest("POST", "/auth/users/unverified-editor/email/verification/resend", nil)
	forbiddenRequest.SetPathValue("id", target.ID)
	forbiddenRequest = forbiddenRequest.WithContext(auth.WithUser(ctx, target))
	forbiddenResponse := httptest.NewRecorder()
	resendRoute.ServeHTTP(forbiddenResponse, forbiddenRequest)
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("non-admin resend status=%d", forbiddenResponse.Code)
	}
	resendRequest := httptest.NewRequest("POST", "/auth/users/unverified-editor/email/verification/resend", nil)
	resendRequest.SetPathValue("id", target.ID)
	resendRequest = resendRequest.WithContext(auth.WithUser(ctx, user))
	resendResponse := httptest.NewRecorder()
	resendRoute.ServeHTTP(resendResponse, resendRequest)
	if resendResponse.Code != http.StatusSeeOther || sentTo != target.Email {
		t.Fatalf("admin resend status=%d email=%q", resendResponse.Code, sentTo)
	}
	resendURL, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	resendVerify := httptest.NewRequest("GET", "/auth/email/verify?token="+url.QueryEscape(resendURL.Query().Get("token")), nil)
	verifiedResponse := httptest.NewRecorder()
	handler.Routes()[1].Handler.ServeHTTP(verifiedResponse, resendVerify)
	loadedTarget, err := users.GetByID(ctx, target.ID)
	if verifiedResponse.Code != http.StatusSeeOther || err != nil || loadedTarget.EmailVerifiedAt == nil {
		t.Fatalf("resent token status=%d user=%+v err=%v", verifiedResponse.Code, loadedTarget, err)
	}
}
