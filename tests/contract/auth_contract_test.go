package contract_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/pquerna/otp/totp"
)

func TestSQLiteAuthContract(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "auth_contract.sqlite")
	runAuthContract(t, database.SQLite, dsn)
}

func TestPostgresAuthContract(t *testing.T) {
	dsn := os.Getenv("HYDRO_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_URL is not set")
	}
	runAuthContract(t, database.Postgres, dsn)
}

func runAuthContract(t *testing.T, dialect database.Dialect, dsn string) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(ctx, dialect, dsn)
	if err != nil {
		t.Fatalf("open %s: %v", dialect, err)
	}
	t.Cleanup(func() { db.Close() })

	if err := database.ApplyMigrations(ctx, db, dialect, migrations.FS); err != nil {
		t.Fatalf("migrate %s: %v", dialect, err)
	}
	resetContractData(t, db, dialect)

	userStore := authstore.NewUserStore(db, dialect)
	sessionStore := authstore.NewSessionStore(db, dialect)
	authSvc := auth.NewAuthService(userStore, sessionStore, auth.NewMobileAuthService(db, dialect), auth.TestArgon2Params)

	// 1. Create User
	u, err := authSvc.CreateUser(ctx, "Admin@HydroGo.io", "Amber-River7!Cedar", auth.RoleAdmin)
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	if u.Email != "admin@hydrogo.io" {
		t.Fatalf("expected normalized lowercase email, got: %s", u.Email)
	}
	verification := auth.EmailVerificationService{Users: userStore, Tokens: userStore}
	var verificationToken string
	verificationTx := database.NewTenantTransactor(db, dialect, false)
	if err := verificationTx.Within(ctx, func(tx database.Executor) error {
		_, token, err := verification.Issue(ctx, tx, u, time.Hour)
		verificationToken = token
		return err
	}); err != nil || verificationToken == "" {
		t.Fatalf("issue verified-email token: token=%t err=%v", verificationToken != "", err)
	}
	if err := verificationTx.Within(ctx, func(tx database.Executor) error { return verification.Verify(ctx, tx, verificationToken) }); err != nil {
		t.Fatalf("verify admin email: %v", err)
	}
	verifiedUser, err := userStore.GetByID(ctx, u.ID)
	if err != nil || verifiedUser.EmailVerifiedAt == nil {
		t.Fatalf("verified admin user=%+v err=%v", verifiedUser, err)
	}

	// 2. Duplicate email rejected
	if _, err := authSvc.CreateUser(ctx, "admin@hydrogo.io", "pass", auth.RoleEditor); err != auth.ErrUserAlreadyExists {
		t.Fatalf("expected ErrUserAlreadyExists, got: %v", err)
	}

	// 3. Authenticate with correct password
	authedUser, err := authSvc.Authenticate(ctx, "ADMIN@hydrogo.io", "Amber-River7!Cedar")
	if err != nil {
		t.Fatalf("auth with valid password failed: %v", err)
	}
	if authedUser.ID != u.ID {
		t.Fatalf("authed user ID mismatch: got %s, want %s", authedUser.ID, u.ID)
	}

	// 4. Authenticate with invalid password
	_, err = authSvc.Authenticate(ctx, "admin@hydrogo.io", "wrong-password")
	if err != auth.ErrInvalidCredentials {
		t.Fatalf("expected ErrInvalidCredentials for wrong password, got: %v", err)
	}

	// 5. Create Session
	rawToken, sess, err := authSvc.CreateSession(ctx, u.ID, "TestAgent/1.0", "127.0.0.1", 1*time.Hour)
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	if rawToken == "" || sess.TokenHash == "" {
		t.Fatalf("expected non-empty rawToken and tokenHash")
	}

	// 6. Validate Session
	valSess, valUser, err := authSvc.ValidateSession(ctx, rawToken)
	if err != nil {
		t.Fatalf("validate session failed: %v", err)
	}
	if valSess.ID != sess.ID || valUser.ID != u.ID {
		t.Fatalf("session or user mismatch")
	}

	// 7. Revoke Session
	if err := authSvc.RevokeSession(ctx, rawToken); err != nil {
		t.Fatalf("revoke session failed: %v", err)
	}

	// 8. Validate revoked session fails
	if _, _, err := authSvc.ValidateSession(ctx, rawToken); err != auth.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound for revoked session, got: %v", err)
	}

	// 9. Disabling a user revokes every session.
	editor, err := authSvc.CreateUser(ctx, "editor@hydrogo.io", "Copper-Lake5!Pine", auth.RoleEditor)
	if err != nil {
		t.Fatalf("create editor failed: %v", err)
	}
	editorToken, _, err := authSvc.CreateSession(ctx, editor.ID, "TestAgent/1.0", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("create editor session failed: %v", err)
	}
	if err := authSvc.SetUserActive(ctx, editor.ID, false); err != nil {
		t.Fatalf("disable editor failed: %v", err)
	}
	if _, _, err := authSvc.ValidateSession(ctx, editorToken); err != auth.ErrSessionNotFound {
		t.Fatalf("expected disabled user session to be revoked, got: %v", err)
	}

	// 10. TOTP and single-use recovery codes are stored safely.
	mfa := auth.MFAService{Store: userStore, MasterKey: []byte("0123456789abcdef0123456789abcdef")}
	var enrollment auth.MFAEnrollment
	err = database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
		var innerErr error
		enrollment, innerErr = mfa.Enroll(ctx, tx, u.ID, u.Email)
		return innerErr
	})
	if err != nil {
		t.Fatalf("MFA enrollment failed: %v", err)
	}
	if len(enrollment.RecoveryCodes) != 8 || enrollment.Secret == "" {
		t.Fatalf("unexpected MFA enrollment: %+v", enrollment)
	}
	if err := database.NewTransactor(db).Within(ctx, func(tx database.Executor) error { return mfa.Enable(ctx, tx, u.ID) }); err != nil {
		t.Fatalf("enable MFA failed: %v", err)
	}
	code, err := totp.GenerateCode(enrollment.Secret, time.Now())
	if err != nil || mfa.Verify(ctx, db, u.ID, code) != nil {
		t.Fatalf("TOTP verification failed: %v", err)
	}
	loginHTTP := auth.HTTPHandler{Service: authSvc, MFA: &mfa, MFADB: db, Transactions: database.NewTransactor(db)}
	loginWithoutMFA := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(url.Values{"email": {u.Email}, "password": {"Amber-River7!Cedar"}}.Encode()))
	loginWithoutMFA.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginWithoutMFAResult := httptest.NewRecorder()
	loginHTTP.Routes()[0].Handler.ServeHTTP(loginWithoutMFAResult, loginWithoutMFA)
	if loginWithoutMFAResult.Code != http.StatusUnauthorized {
		t.Fatalf("login without MFA status = %d", loginWithoutMFAResult.Code)
	}
	loginWithMFA := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(url.Values{"email": {u.Email}, "password": {"Amber-River7!Cedar"}, "mfa_code": {code}}.Encode()))
	loginWithMFA.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginWithMFAResult := httptest.NewRecorder()
	loginHTTP.Routes()[0].Handler.ServeHTTP(loginWithMFAResult, loginWithMFA)
	if loginWithMFAResult.Code != http.StatusOK {
		t.Fatalf("login with MFA status = %d", loginWithMFAResult.Code)
	}
	recovery := enrollment.RecoveryCodes[0]
	if err := database.NewTransactor(db).Within(ctx, func(tx database.Executor) error { return mfa.ConsumeRecoveryCode(ctx, tx, u.ID, recovery) }); err != nil {
		t.Fatalf("recovery code consume failed: %v", err)
	}
	err = database.NewTransactor(db).Within(ctx, func(tx database.Executor) error { return mfa.ConsumeRecoveryCode(ctx, tx, u.ID, recovery) })
	if !errors.Is(err, auth.ErrMFAInvalidCode) {
		t.Fatalf("second recovery code consume = %v, want invalid code", err)
	}

	// 11. Password reset tokens are purpose-bound, short-lived and single-use.
	resetService := auth.PasswordRecovery{Users: userStore, Tokens: userStore, Sessions: sessionStore, ArgonParams: auth.TestArgon2Params}
	var resetToken string
	resetToken, err = resetService.Issue(ctx, db, u.Email, time.Minute)
	if err != nil {
		t.Fatalf("issue reset token failed: %v", err)
	}
	err = database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
		return resetService.Reset(ctx, tx, resetToken, "Copper-Lake5!Pine")
	})
	if err != nil {
		t.Fatalf("reset password failed: %v", err)
	}
	if _, err := authSvc.Authenticate(ctx, u.Email, "Copper-Lake5!Pine"); err != nil {
		t.Fatalf("new password authentication failed: %v", err)
	}
	if _, err := authSvc.Authenticate(ctx, u.Email, "Amber-River7!Cedar"); err != auth.ErrInvalidCredentials {
		t.Fatalf("old password result = %v, want invalid credentials", err)
	}
	if err := database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
		return resetService.Reset(ctx, tx, resetToken, "Maple-Cloud8!Birch")
	}); !errors.Is(err, auth.ErrRecoveryTokenInvalid) {
		t.Fatalf("second reset result = %v, want invalid token", err)
	}

	// 12. Invitations are role-limited, hashed and single-use.
	invitation := auth.InvitationService{Users: userStore, Invites: userStore, ArgonParams: auth.TestArgon2Params}
	inviteToken, err := invitation.Issue(ctx, db, "new-editor@hydrogo.io", auth.RoleEditor, time.Hour)
	if err != nil {
		t.Fatalf("issue invitation failed: %v", err)
	}
	var invited *auth.User
	err = database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
		var inner error
		invited, inner = invitation.Accept(ctx, tx, inviteToken, "Gold-Harbor9!Willow")
		return inner
	})
	if err != nil || invited.Role != auth.RoleEditor {
		t.Fatalf("accept invitation = %#v/%v", invited, err)
	}
	if invited.EmailVerifiedAt == nil {
		t.Fatal("accepted email invitation did not verify the invited address")
	}
	if _, err := authSvc.Authenticate(ctx, invited.Email, "Gold-Harbor9!Willow"); err != nil {
		t.Fatalf("invited user authentication failed: %v", err)
	}
	if err := database.NewTransactor(db).Within(ctx, func(tx database.Executor) error {
		_, inner := invitation.Accept(ctx, tx, inviteToken, "Maple-Cloud8!Birch")
		return inner
	}); !errors.Is(err, auth.ErrInviteInvalid) {
		t.Fatalf("second invitation acceptance = %v, want invalid invitation", err)
	}
	unverified := &auth.User{ID: "change-email-user", Email: "before@example.test", PasswordHash: "unused", Role: auth.RoleAdmin, Active: true}
	if err := userStore.Create(ctx, unverified); err != nil {
		t.Fatal(err)
	}
	var pendingToken string
	if err := verificationTx.Within(ctx, func(tx database.Executor) error {
		_, token, issueErr := verification.Issue(ctx, tx, unverified, time.Hour)
		pendingToken = token
		return issueErr
	}); err != nil {
		t.Fatal(err)
	}
	if err := verificationTx.Within(ctx, func(tx database.Executor) error {
		return userStore.UpdateEmail(ctx, tx, unverified.ID, "after@example.test")
	}); err != nil {
		t.Fatal(err)
	}
	if err := verificationTx.Within(ctx, func(tx database.Executor) error { return verification.Verify(ctx, tx, pendingToken) }); !errors.Is(err, auth.ErrEmailVerificationInvalid) {
		t.Fatalf("old address verification token = %v, want invalid", err)
	}
}
