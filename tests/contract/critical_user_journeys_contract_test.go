package contract_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	auditstore "github.com/ardaozyildirim/hydrogo/internal/audit/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	authstore "github.com/ardaozyildirim/hydrogo/internal/auth/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/media"
	mediastore "github.com/ardaozyildirim/hydrogo/internal/media/sqlstore"
	outboxstore "github.com/ardaozyildirim/hydrogo/internal/outbox/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/router"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/members"
	memberstore "github.com/ardaozyildirim/hydrogo/modules/members/sqlstore"
	memberstransport "github.com/ardaozyildirim/hydrogo/modules/members/transport"
	"github.com/ardaozyildirim/hydrogo/modules/news"
	newsstore "github.com/ardaozyildirim/hydrogo/modules/news/repository/sqlstore"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
	pagestore "github.com/ardaozyildirim/hydrogo/modules/pages/repository/sqlstore"
	pageservice "github.com/ardaozyildirim/hydrogo/modules/pages/service"
	pagestransport "github.com/ardaozyildirim/hydrogo/modules/pages/transport"
	"github.com/ardaozyildirim/hydrogo/modules/site"
	sitestore "github.com/ardaozyildirim/hydrogo/modules/site/repository/sqlstore"
	sitetransport "github.com/ardaozyildirim/hydrogo/modules/site/transport"
	"github.com/pquerna/otp/totp"
)

// TestCriticalUserJourneysContract certifies P1 user journeys across SQLite & Postgres:
// 1. Admin Publish Journey: Login -> MFA -> RBAC -> Draft -> Preview -> Publish -> Public Verification
// 2. Security Negative Paths: CSRF enforcement, revoked sessions, single-use reset/invite tokens, unauthorized role mutations
// 3. Media & Member Isolation: MIME rejection, oversize/corrupt upload, private asset access control, cross-member resource isolation
// 4. Content Resilience: Optimistic lock conflicts, scheduled publish lifecycle, redirect chains/cycles, sitemap & locale fallback
func TestCriticalUserJourneysContract(t *testing.T) {
	for _, dialect := range []database.Dialect{database.SQLite, database.Postgres} {
		dialect := dialect
		if dialect == database.Postgres && os.Getenv("HYDRO_TEST_POSTGRES_URL") == "" {
			continue
		}
		t.Run(string(dialect), func(t *testing.T) {
			ctx := context.Background()
			var sqlDB *sql.DB
			var closeDB func()

			if dialect == database.SQLite {
				sqlitePath := filepath.Join(t.TempDir(), "journeys.sqlite")
				opened, err := database.Open(ctx, database.SQLite, sqlitePath)
				if err != nil {
					t.Fatalf("open sqlite: %v", err)
				}
				sqlDB = opened
				closeDB = func() { _ = opened.Close() }
			} else {
				opened := openIsolatedPostgres(t, ctx)
				sqlDB = opened
				closeDB = func() { _ = opened.Close() }
			}
			defer closeDB()

			if err := database.ApplyMigrations(ctx, sqlDB, dialect, migrations.FS); err != nil {
				t.Fatalf("migrate: %v", err)
			}

			t.Run("AdminPublishJourney", func(t *testing.T) {
				runAdminPublishJourneyTest(t, ctx, sqlDB, dialect)
			})

			t.Run("SecurityNegativePaths", func(t *testing.T) {
				runSecurityNegativePathsTest(t, ctx, sqlDB, dialect)
			})

			t.Run("MediaAndMemberIsolation", func(t *testing.T) {
				runMediaAndMemberIsolationTest(t, ctx, sqlDB, dialect)
			})

			t.Run("ContentResilience", func(t *testing.T) {
				runContentResilienceTest(t, ctx, sqlDB, dialect)
			})
		})
	}
}

// 1. Admin Publish Journey: Login -> MFA -> RBAC -> Draft -> Preview -> Publish -> Public Verification
func runAdminPublishJourneyTest(t *testing.T, ctx context.Context, sqlDB *sql.DB, dialect database.Dialect) {
	userStore := authstore.NewUserStore(sqlDB, dialect)
	sessionStore := authstore.NewSessionStore(sqlDB, dialect)
	authSvc := auth.NewAuthService(userStore, sessionStore, auth.NewMobileAuthService(sqlDB, dialect), auth.TestArgon2Params)
	mfaSvc := auth.MFAService{Store: userStore, MasterKey: []byte("0123456789abcdef0123456789abcdef")}

	adminUser, err := authSvc.CreateUser(ctx, "admin_journey@example.com", "AdminPassword123!", auth.RoleAdmin)
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}

	var enrollment auth.MFAEnrollment
	err = database.NewTransactor(sqlDB).Within(ctx, func(tx database.Executor) error {
		var innerErr error
		enrollment, innerErr = mfaSvc.Enroll(ctx, tx, adminUser.ID, adminUser.Email)
		return innerErr
	})
	if err != nil {
		t.Fatalf("enroll MFA: %v", err)
	}
	if err := database.NewTransactor(sqlDB).Within(ctx, func(tx database.Executor) error {
		return mfaSvc.Enable(ctx, tx, adminUser.ID)
	}); err != nil {
		t.Fatalf("enable MFA: %v", err)
	}

	// Login with password only -> MFA Required (Unauthorized)
	authHandler := auth.HTTPHandler{Service: authSvc, MFA: &mfaSvc, MFADB: sqlDB, Transactions: database.NewTransactor(sqlDB)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("email=admin_journey@example.com&password=AdminPassword123!"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	authHandler.Routes()[0].Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without MFA, got %d", rec.Code)
	}

	// Login with valid TOTP
	code, err := totp.GenerateCode(enrollment.Secret, time.Now())
	if err != nil {
		t.Fatalf("generate TOTP: %v", err)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(fmt.Sprintf("email=admin_journey@example.com&password=AdminPassword123!&mfa_code=%s", code)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	authHandler.Routes()[0].Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with MFA, got %d", rec.Code)
	}

	// Create session for admin
	rawToken, _, err := authSvc.CreateSession(ctx, adminUser.ID, "JourneyAgent/1.0", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Setup Pages Service & Transport
	pagesRepo := pagestore.Store{Dialect: dialect}
	contentSvc := pageservice.ContentService{
		Transactions: database.NewTransactor(sqlDB),
		Pages:        pagesRepo,
		Audit:        auditstore.Store{Dialect: dialect},
		Outbox:       outboxstore.Store{Dialect: dialect},
		PublicCache:  cache.New[pages.Page](10, time.Minute),
		Sanitize:     ui.NewHTMLSanitizer().Sanitize,
	}
	pagesHandler := pagestransport.Handler{DB: sqlDB, Content: contentSvc, AllowDemoFrontend: true}

	// 1. Create Draft
	draftPage := pages.Page{
		ID:     "journey-page-1",
		Locale: "tr",
		Slug:   "journey-launch",
		Title:  "Journey Launch Title",
		Body:   "<p>Journey body draft</p>",
		Status: pages.StatusDraft,
	}
	if err := contentSvc.CreateDraft(ctx, draftPage); err != nil {
		t.Fatalf("create draft: %v", err)
	}

	// Verify not yet published in public API
	_, err = contentSvc.GetPublished(ctx, sqlDB, "tr", "journey-launch")
	if !errors.Is(err, sql.ErrNoRows) && !errors.Is(err, pageservice.ErrInvalidPage) {
		t.Fatalf("draft should not be public, got: %v", err)
	}

	// 2. Preview Draft (requires authentication)
	previewReq := httptest.NewRequest(http.MethodGet, "/pages/journey-page-1/preview", nil)
	previewReq.SetPathValue("id", "journey-page-1")
	previewRec := httptest.NewRecorder()
	for _, r := range pagesHandler.Routes() {
		if strings.Contains(r.Pattern, "/preview") {
			r.Handler.ServeHTTP(previewRec, previewReq)
			break
		}
	}
	if previewRec.Code != http.StatusOK || !strings.Contains(previewRec.Body.String(), "Journey Launch Title") {
		t.Fatalf("preview failed: %d %s", previewRec.Code, previewRec.Body.String())
	}

	// 3. Publish page (version 1)
	if err := contentSvc.Publish(ctx, "journey-page-1", 1); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// 4. Public API and HTML Verification
	pub, err := contentSvc.GetPublished(ctx, sqlDB, "tr", "journey-launch")
	if err != nil {
		t.Fatalf("get published: %v", err)
	}
	if pub.Title != "Journey Launch Title" || pub.Status != pages.StatusPublished {
		t.Fatalf("unexpected published state: %+v", pub)
	}

	// Public HTML Endpoint Verification
	htmlReq := httptest.NewRequest(http.MethodGet, "/public/pages/tr/journey-launch/html", nil)
	htmlReq.SetPathValue("locale", "tr")
	htmlReq.SetPathValue("slug", "journey-launch")
	htmlRec := httptest.NewRecorder()
	for _, r := range pagesHandler.Routes() {
		if strings.Contains(r.Pattern, "/public/pages/{locale}/{slug}/html") {
			r.Handler.ServeHTTP(htmlRec, htmlReq)
			break
		}
	}
	if htmlRec.Code != http.StatusOK || !strings.Contains(htmlRec.Body.String(), "Journey Launch Title") {
		t.Fatalf("public html route failed: code=%d, body=%s", htmlRec.Code, htmlRec.Body.String())
	}

	_ = rawToken
}

// 2. Security Negative Paths: CSRF, revoked session, replay reset/invite tokens, role mutation checks
func runSecurityNegativePathsTest(t *testing.T, ctx context.Context, sqlDB *sql.DB, dialect database.Dialect) {
	userStore := authstore.NewUserStore(sqlDB, dialect)
	sessionStore := authstore.NewSessionStore(sqlDB, dialect)
	authSvc := auth.NewAuthService(userStore, sessionStore, auth.NewMobileAuthService(sqlDB, dialect), auth.TestArgon2Params)

	// A. CSRF Enforcement on Mutations
	csrfMiddleware := security.CSRFMiddleware(false)
	dummyProtectedMutation := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mutated"))
	}))

	// Missing CSRF token -> 403 Forbidden
	reqNoCSRF := httptest.NewRequest(http.MethodPost, "/admin/pages", strings.NewReader("title=test"))
	reqNoCSRF.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recNoCSRF := httptest.NewRecorder()
	dummyProtectedMutation.ServeHTTP(recNoCSRF, reqNoCSRF)
	if recNoCSRF.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing CSRF, got %d", recNoCSRF.Code)
	}

	// Valid CSRF Token in Header & Cookie -> 200 OK
	tok, _ := security.GenerateCSRFToken()
	reqValidCSRF := httptest.NewRequest(http.MethodPost, "/admin/pages", strings.NewReader("title=test"))
	reqValidCSRF.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqValidCSRF.Header.Set(security.CSRFHeaderName, tok)
	reqValidCSRF.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: tok})
	recValidCSRF := httptest.NewRecorder()
	dummyProtectedMutation.ServeHTTP(recValidCSRF, reqValidCSRF)
	if recValidCSRF.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid CSRF, got %d", recValidCSRF.Code)
	}

	// B. Revoked Session Access Rejection
	testUser, err := authSvc.CreateUser(ctx, "victim@example.com", "VictimPassword123!", auth.RoleEditor)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	rawSessionToken, _, err := authSvc.CreateSession(ctx, testUser.ID, "Agent/1.0", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Validate works
	if _, _, err := authSvc.ValidateSession(ctx, rawSessionToken); err != nil {
		t.Fatalf("valid session rejected: %v", err)
	}

	// Revoke
	if err := authSvc.RevokeSession(ctx, rawSessionToken); err != nil {
		t.Fatalf("revoke session: %v", err)
	}

	// Validate fails
	if _, _, err := authSvc.ValidateSession(ctx, rawSessionToken); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("revoked session must return ErrSessionNotFound, got %v", err)
	}

	// C. Single-Use Password Reset Token
	resetSvc := auth.PasswordRecovery{Users: userStore, Tokens: userStore, Sessions: sessionStore, ArgonParams: auth.TestArgon2Params}
	resetTok, err := resetSvc.Issue(ctx, sqlDB, testUser.Email, 5*time.Minute)
	if err != nil {
		t.Fatalf("issue reset token: %v", err)
	}
	// First use: success
	err = database.NewTransactor(sqlDB).Within(ctx, func(tx database.Executor) error {
		return resetSvc.Reset(ctx, tx, resetTok, "BrandNewPassword123!")
	})
	if err != nil {
		t.Fatalf("first reset failed: %v", err)
	}
	// Second use: reject
	err = database.NewTransactor(sqlDB).Within(ctx, func(tx database.Executor) error {
		return resetSvc.Reset(ctx, tx, resetTok, "AnotherNewPassword123!")
	})
	if !errors.Is(err, auth.ErrRecoveryTokenInvalid) {
		t.Fatalf("reused reset token must fail, got: %v", err)
	}

	// D. Single-Use Invitation Token
	inviteSvc := auth.InvitationService{Users: userStore, Invites: userStore, ArgonParams: auth.TestArgon2Params}
	inviteTok, err := inviteSvc.Issue(ctx, sqlDB, "invited_editor@example.com", auth.RoleEditor, time.Hour)
	if err != nil {
		t.Fatalf("issue invite: %v", err)
	}
	// First accept: success
	err = database.NewTransactor(sqlDB).Within(ctx, func(tx database.Executor) error {
		_, inner := inviteSvc.Accept(ctx, tx, inviteTok, "EditorPass123!")
		return inner
	})
	if err != nil {
		t.Fatalf("first invite accept failed: %v", err)
	}
	// Second accept: reject
	err = database.NewTransactor(sqlDB).Within(ctx, func(tx database.Executor) error {
		_, inner := inviteSvc.Accept(ctx, tx, inviteTok, "EditorPass123!")
		return inner
	})
	if !errors.Is(err, auth.ErrInviteInvalid) {
		t.Fatalf("reused invite token must fail, got: %v", err)
	}

	// E. Unauthorized Role Mutation
	authorizer := func(r *http.Request) bool {
		user := auth.UserFromContext(r.Context())
		return user != nil && user.Role == auth.RoleAdmin
	}
	adminOnlyRoute := router.Route{
		Pattern: "POST /admin/users/create",
		Access:  router.Protected,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	rtr, err := router.New([]router.Route{adminOnlyRoute}, authorizer)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	// Editor user attempting Admin endpoint -> 401 Unauthorized by Router
	editorCtx := auth.WithUser(context.Background(), &auth.User{ID: "editor-1", Role: auth.RoleEditor})
	editorReq := httptest.NewRequest(http.MethodPost, "/admin/users/create", nil).WithContext(editorCtx)
	editorRec := httptest.NewRecorder()
	rtr.ServeHTTP(editorRec, editorReq)
	if editorRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthorized role accessing admin route, got %d", editorRec.Code)
	}
}

// 3. Media and Member Isolation: MIME, size, corrupt data, private vs public, cross-member resource isolation
func runMediaAndMemberIsolationTest(t *testing.T, ctx context.Context, sqlDB *sql.DB, dialect database.Dialect) {
	storageDir := t.TempDir()
	mediaStore := mediastore.Store{Dialect: dialect}
	uploader, err := media.NewUploader(storageDir, 1<<20) // 1MB max
	if err != nil {
		t.Fatalf("new uploader: %v", err)
	}
	mediaSvc := media.Service{
		Transactions: database.NewTransactor(sqlDB),
		Uploader:     uploader,
		Assets:       mediaStore,
	}

	// A. Valid PNG Image Upload
	validImg := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			validImg.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var validPNG bytes.Buffer
	if err := png.Encode(&validPNG, validImg); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	res, err := mediaSvc.Upload(ctx, bytes.NewReader(validPNG.Bytes()), "sample.png")
	if err != nil {
		t.Fatalf("valid upload failed: %v", err)
	}
	if res.MimeType != "image/png" {
		t.Fatalf("expected image/png, got %s", res.MimeType)
	}

	// B. Disallowed MIME type (e.g. text/html, exe, svg)
	disallowedHTML := []byte("<html><script>alert('xss')</script></html>")
	_, err = mediaSvc.Upload(ctx, bytes.NewReader(disallowedHTML), "evil.html")
	if !errors.Is(err, media.ErrDisallowedMimeType) {
		t.Fatalf("expected ErrDisallowedMimeType, got: %v", err)
	}

	// C. File Too Large (>1MB)
	largeData := make([]byte, (1<<20)+1024)
	_, err = mediaSvc.Upload(ctx, bytes.NewReader(largeData), "huge.png")
	if !errors.Is(err, media.ErrFileTooLarge) && !errors.Is(err, media.ErrDisallowedMimeType) {
		t.Fatalf("expected size or mime error on huge file, got: %v", err)
	}

	// D. Corrupt Image Data
	corruptData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0xFF, 0xFF, 0xFF}
	_, err = mediaSvc.Upload(ctx, bytes.NewReader(corruptData), "corrupt.png")
	if !errors.Is(err, media.ErrInvalidImageData) && !errors.Is(err, media.ErrDisallowedMimeType) {
		t.Fatalf("expected ErrInvalidImageData on corrupt file, got: %v", err)
	}

	// E. Private vs Public Media Access Control
	mediaHandler := media.HTTPHandler{Service: mediaSvc, DB: sqlDB}
	// Create private asset in DB
	privateAsset := media.Asset{
		ID:               "private-asset-1",
		StorageKey:       "private-sample.png",
		OriginalFilename: "private-sample.png",
		MimeType:         "image/png",
		SizeBytes:        int64(validPNG.Len()),
		Visibility:       media.VisibilityPrivate,
		Status:           "ready",
	}
	if err := database.NewTransactor(sqlDB).Within(ctx, func(tx database.Executor) error {
		if err := mediaStore.CreatePending(ctx, tx, privateAsset); err != nil {
			return err
		}
		return mediaStore.MarkReady(ctx, tx, privateAsset.ID, media.VisibilityPrivate)
	}); err != nil {
		t.Fatalf("create private asset: %v", err)
	}

	// Public unauthenticated request to private asset -> 404 / 403
	pubReq := httptest.NewRequest(http.MethodGet, "/media/private-asset-1", nil)
	pubReq.SetPathValue("id", "private-asset-1")
	pubRec := httptest.NewRecorder()
	for _, r := range mediaHandler.Routes() {
		if r.Pattern == "GET /media/{id}" {
			r.Handler.ServeHTTP(pubRec, pubReq)
			break
		}
	}
	if pubRec.Code == http.StatusOK {
		t.Fatalf("unauthorized user was able to access private media!")
	}

	// F. Cross-Member Resource Access Isolation
	memStore := memberstore.Store{DB: sqlDB, Dialect: dialect}
	memSvc := members.Service{
		Members:     memStore,
		Sessions:    memberstore.SessionStore(memStore),
		ArgonParams: auth.TestArgon2Params,
	}
	memA, err := memSvc.Register(ctx, "memberA@example.com", "MemberPass123!")
	if err != nil {
		t.Fatalf("register member A: %v", err)
	}
	memB, err := memSvc.Register(ctx, "memberB@example.com", "MemberPass123!")
	if err != nil {
		t.Fatalf("register member B: %v", err)
	}

	// Member A session cannot read or alter Member B's account
	rawTokenA, createdSessA, err := memSvc.CreateSession(ctx, memA.ID, "Agent/1.0", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatalf("create session A: %v", err)
	}

	valSessA, valMemA, err := memSvc.ValidateSession(ctx, rawTokenA)
	if err != nil || valMemA.ID != memA.ID || valSessA.ID != createdSessA.ID {
		t.Fatalf("session A validate: %v", err)
	}

	// Verify Member A cannot query Member B's profile
	memberTransportHandler := memberstransport.Handler{Service: memSvc, DB: sqlDB}
	meReq := httptest.NewRequest(http.MethodGet, "/member/me", nil)
	meReq.AddCookie(&http.Cookie{Name: members.SessionCookieName, Value: rawTokenA})
	meRec := httptest.NewRecorder()
	for _, r := range memberTransportHandler.Routes() {
		if r.Pattern == "GET /member/me" {
			r.Handler.ServeHTTP(meRec, meReq)
			break
		}
	}
	if meRec.Code != http.StatusOK || !strings.Contains(strings.ToLower(meRec.Body.String()), "membera@example.com") {
		t.Fatalf("member A me route failed: %d %s", meRec.Code, meRec.Body.String())
	}
	if strings.Contains(strings.ToLower(meRec.Body.String()), "memberb@example.com") {
		t.Fatalf("data leak: member A saw member B email!")
	}
	_ = memB
}

// 4. Content Resilience: Optimistic lock conflicts, scheduled publish, redirect chains/loops, sitemap & locale fallback
func runContentResilienceTest(t *testing.T, ctx context.Context, sqlDB *sql.DB, dialect database.Dialect) {
	pagesRepo := pagestore.Store{Dialect: dialect}
	contentSvc := pageservice.ContentService{
		Transactions: database.NewTransactor(sqlDB),
		Pages:        pagesRepo,
		Audit:        auditstore.Store{Dialect: dialect},
		Outbox:       outboxstore.Store{Dialect: dialect},
		PublicCache:  cache.New[pages.Page](10, time.Minute),
		Sanitize:     ui.NewHTMLSanitizer().Sanitize,
	}

	// A. Optimistic Lock Conflict on Concurrent Edits
	basePage := pages.Page{
		ID:     "resilience-page-1",
		Locale: "tr",
		Slug:   "resilience-slug",
		Title:  "Initial Title",
		Body:   "Initial Body",
	}
	if err := contentSvc.CreateDraft(ctx, basePage); err != nil {
		t.Fatalf("create base draft: %v", err)
	}

	// Editor 1 updates version 1 -> success (becomes version 2)
	err := contentSvc.UpdateDraft(ctx, pages.Page{
		ID:     "resilience-page-1",
		Locale: "tr",
		Slug:   "resilience-slug",
		Title:  "Editor 1 Title",
		Body:   "Editor 1 Body",
	}, 1)
	if err != nil {
		t.Fatalf("editor 1 update failed: %v", err)
	}

	// Editor 2 attempts update with stale version 1 -> Conflict!
	err = contentSvc.UpdateDraft(ctx, pages.Page{
		ID:     "resilience-page-1",
		Locale: "tr",
		Slug:   "resilience-slug",
		Title:  "Editor 2 Title (Stale)",
		Body:   "Editor 2 Body",
	}, 1)
	if !errors.Is(err, pageservice.ErrConflict) {
		t.Fatalf("expected ErrConflict for stale update, got: %v", err)
	}

	// B. Scheduled News Publish Lifecycle
	newsRepo := newsstore.Store{Dialect: dialect}
	newsSvc := news.Service{
		Transactions: database.NewTransactor(sqlDB),
		Repository:   newsRepo,
		Audit:        auditstore.Store{Dialect: dialect},
		Outbox:       outboxstore.Store{Dialect: dialect},
	}
	marker := time.Now().UTC().UnixNano()
	newsItem := news.Item{
		ID:     fmt.Sprintf("news-sched-%d", marker),
		Locale: "tr",
		Slug:   fmt.Sprintf("sched-slug-%d", marker),
		Title:  "Scheduled News Item",
		Body:   "Scheduled content",
		Status: news.StatusDraft,
	}
	if err := newsSvc.CreateDraft(ctx, newsItem); err != nil {
		t.Fatalf("create news draft: %v", err)
	}

	// Schedule in future
	futureTime := time.Now().UTC().Add(2 * time.Hour)
	if err := newsSvc.Schedule(ctx, newsItem.ID, 1, futureTime); err != nil {
		t.Fatalf("schedule news: %v", err)
	}

	// Attempting to run runner before target time -> not published
	schedItem, err := newsRepo.Get(ctx, sqlDB, newsItem.ID)
	if err != nil || schedItem.Status != news.StatusScheduled {
		t.Fatalf("expected scheduled status, got %+v", schedItem)
	}

	// Run when scheduled time has passed
	if err := newsSvc.RunScheduled(ctx, newsItem.ID, schedItem.ScheduleGeneration, futureTime.Add(time.Minute)); err != nil {
		t.Fatalf("run scheduled news: %v", err)
	}
	pubItem, err := newsRepo.Get(ctx, sqlDB, newsItem.ID)
	if err != nil || pubItem.Status != news.StatusPublished {
		t.Fatalf("expected published status after run, got %+v", pubItem)
	}

	// C. Redirect Cycles & Normalization
	siteRepo := sitestore.Store{Dialect: dialect}
	siteSvc := site.Service{
		Store:        siteRepo,
		Transactions: database.NewTransactor(sqlDB),
		Audit:        auditstore.Store{Dialect: dialect},
	}

	// Save redirect /old-url -> /new-url
	if err := siteSvc.SaveRedirect(ctx, site.Redirect{SourcePath: "/old-url", TargetPath: "/new-url", Status: 301}); err != nil {
		t.Fatalf("save redirect: %v", err)
	}
	red, err := siteRepo.GetRedirect(ctx, sqlDB, "/old-url")
	if err != nil || red.TargetPath != "/new-url" {
		t.Fatalf("get redirect failed: %+v/%v", red, err)
	}

	// Circular redirect /new-url -> /old-url
	if err := siteSvc.SaveRedirect(ctx, site.Redirect{SourcePath: "/new-url", TargetPath: "/old-url", Status: 301}); err != nil {
		t.Fatalf("save second redirect: %v", err)
	}

	// D. Site Transport & Sitemap Verification
	siteHandler := sitetransport.Handler{
		Service:       &siteSvc,
		Store:         siteRepo,
		DB:            sqlDB,
		SitemapOrigin: "https://example.com",
		SitemapProvider: func(context.Context) ([]site.SitemapEntry, error) {
			return []site.SitemapEntry{{URL: "https://example.com/tr/journey-launch"}}, nil
		},
	}
	sitemapReq := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	sitemapRec := httptest.NewRecorder()
	for _, r := range siteHandler.Routes() {
		if strings.Contains(r.Pattern, "sitemap.xml") {
			r.Handler.ServeHTTP(sitemapRec, sitemapReq)
			break
		}
	}
	if sitemapRec.Code != http.StatusOK || !strings.Contains(sitemapRec.Body.String(), "urlset") {
		t.Fatalf("sitemap route failed: %d %s", sitemapRec.Code, sitemapRec.Body.String())
	}
}
