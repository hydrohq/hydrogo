package transport

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/testhttp"
	"github.com/ardaozyildirim/hydrogo/modules/members"
	"github.com/ardaozyildirim/hydrogo/modules/members/sqlstore"
)

func setupOAuthHandlerDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "member-oauth-http.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	schema := `
	CREATE TABLE members (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		active INTEGER NOT NULL DEFAULT 1,
		email_verified_at TEXT,
		last_login_at TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		tenant_id TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE member_sessions (
		id TEXT PRIMARY KEY,
		member_id TEXT NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		expires_at TEXT NOT NULL,
		created_at TEXT NOT NULL,
		last_seen_at TEXT NOT NULL,
		user_agent TEXT NOT NULL,
		ip_address TEXT NOT NULL,
		tenant_id TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE member_oauth_identities (
		provider TEXT NOT NULL,
		subject TEXT NOT NULL,
		member_id TEXT NOT NULL,
		tenant_id TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		PRIMARY KEY (provider, subject, tenant_id)
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	return db
}

// TestMemberOAuthStartAndCallbackProvisionsMemberAndIssuesSession mirrors
// internal/auth's TestSocialOAuthStartAndCallbackUseStatePKCEAndCreateOpaqueSession
// on the member-facing surface: it exercises the full PKCE + signed-state
// dance through the shared auth.OAuthFlow, but asserts the member-specific
// outcome — a brand-new member is auto-provisioned (there is no account to
// link to yet) and the member session cookie, not the admin one, is issued.
func TestMemberOAuthStartAndCallbackProvisionsMemberAndIssuesSession(t *testing.T) {
	db := setupOAuthHandlerDB(t)
	service := members.Service{Members: sqlstore.Store{DB: db, Dialect: database.SQLite}, Sessions: sqlstore.SessionStore{DB: db, Dialect: database.SQLite}, ArgonParams: auth.DefaultArgon2Params}
	resolver := sqlstore.OAuthStore{DB: db, Dialect: database.SQLite}

	var verifier string
	const upstream = "http://127.0.0.1:19096"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19096": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			verifier = r.Form.Get("code_verifier")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"upstream-token"}`))
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer upstream-token" {
				t.Errorf("auth header=%q", r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"sub":"google-member-subject","email":"member@example.test","email_verified":true,"name":"Member"}`))
		default:
			http.NotFound(w, r)
		}
	})})

	provider := auth.OAuthProviderConfig{Name: "google", ClientID: "client", ClientSecret: "secret", AuthorizationURL: upstream + "/authorize", TokenURL: upstream + "/token", UserInfoURL: upstream + "/userinfo", RedirectURL: upstream + "/callback", Scopes: []string{"openid", "email"}}
	handler := OAuthHandler{
		OAuthFlow: auth.OAuthFlow{
			Providers:              map[string]auth.OAuthProviderConfig{"google": provider},
			HTTPClient:             client,
			StateKey:               []byte(strings.Repeat("s", 32)),
			AllowInsecureLocalhost: true,
			CookieName:             "hydro_member_oauth_state",
			CookiePath:             "/member/oauth/",
			DefaultReturnTo:        "/member/account",
			CancelRedirect:         "/member/login?social=cancelled",
		},
		Resolver: resolver,
		Service:  service,
	}

	start := httptest.NewRecorder()
	startRequest := httptest.NewRequest(http.MethodGet, "/member/oauth/google/start?return_to=%2Fmember%2Faccount", nil)
	startRequest.SetPathValue("provider", "google")
	handler.Start(start, startRequest)
	if start.Code != http.StatusFound {
		t.Fatalf("start status=%d", start.Code)
	}
	authorize, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state := authorize.Query().Get("state")
	cookie := start.Result().Cookies()[0]

	callback := httptest.NewRequest(http.MethodGet, "/member/oauth/google/callback?state="+url.QueryEscape(state)+"&code=once", nil)
	callback.SetPathValue("provider", "google")
	callback.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.callback(response, callback)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/member/account" {
		t.Fatalf("callback status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
	if verifier == "" {
		t.Fatal("PKCE verifier was not sent to the token endpoint")
	}

	var sessionCookie *http.Cookie
	for _, item := range response.Result().Cookies() {
		if item.Name == members.SessionCookieName && item.Value != "" {
			sessionCookie = item
		}
	}
	if sessionCookie == nil {
		t.Fatal("member session cookie was not issued")
	}

	member, err := sqlstore.Store{DB: db, Dialect: database.SQLite}.GetByEmail(callback.Context(), "member@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if !member.Active || member.PasswordHash != "" || member.EmailVerifiedAt == nil {
		t.Fatalf("auto-provisioned member = %+v", member)
	}
}

// TestMemberOAuthRoutesRequireConfiguration mirrors the admin handler's
// Routes() guard: a handler missing providers, a resolver or a state key
// registers no routes rather than serving a broken social login flow.
func TestMemberOAuthRoutesRequireConfiguration(t *testing.T) {
	if routes := (OAuthHandler{}).Routes(); routes != nil {
		t.Fatalf("unconfigured handler returned routes: %+v", routes)
	}
	configured := OAuthHandler{
		OAuthFlow: auth.OAuthFlow{Providers: map[string]auth.OAuthProviderConfig{"google": {}}, StateKey: []byte(strings.Repeat("k", 32))},
		Resolver:  sqlstore.OAuthStore{},
	}
	if routes := configured.Routes(); len(routes) != 2 {
		t.Fatalf("configured handler routes = %d, want 2", len(routes))
	}
}
