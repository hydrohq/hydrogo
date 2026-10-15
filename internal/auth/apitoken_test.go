package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/security"
)

type memoryTokenStore struct{ tokens map[string]*APIToken }

func newMemoryTokenStore() *memoryTokenStore {
	return &memoryTokenStore{tokens: map[string]*APIToken{}}
}

func (m *memoryTokenStore) Create(_ context.Context, token *APIToken) error {
	copied := *token
	m.tokens[token.ID] = &copied
	return nil
}
func (m *memoryTokenStore) ListForUser(_ context.Context, userID string) ([]APIToken, error) {
	var items []APIToken
	for _, token := range m.tokens {
		if token.UserID == userID && token.Kind == APITokenPersonal && token.RevokedAt == nil {
			items = append(items, *token)
		}
	}
	return items, nil
}
func (m *memoryTokenStore) ListAPIKeys(context.Context) ([]APIToken, error) {
	var items []APIToken
	for _, token := range m.tokens {
		if token.Kind == APITokenKey && token.RevokedAt == nil {
			items = append(items, *token)
		}
	}
	return items, nil
}
func (m *memoryTokenStore) Revoke(_ context.Context, id, userID string) error {
	token, ok := m.tokens[id]
	if !ok || token.RevokedAt != nil || (userID != "" && token.UserID != userID) {
		return ErrAPITokenNotFound
	}
	now := time.Now().UTC()
	token.RevokedAt = &now
	return nil
}
func (m *memoryTokenStore) FindByHash(_ context.Context, hash string) (*APIToken, error) {
	for _, token := range m.tokens {
		if token.TokenHash == hash {
			copied := *token
			return &copied, nil
		}
	}
	return nil, ErrAPITokenNotFound
}
func (m *memoryTokenStore) TouchLastUsed(_ context.Context, id string, at time.Time) error {
	if token, ok := m.tokens[id]; ok {
		token.LastUsedAt = &at
	}
	return nil
}

func newTokenTestService(t *testing.T, user *User) (*AuthService, *memoryTokenStore) {
	t.Helper()
	store := newMemoryTokenStore()
	service := NewAuthService(httpUserRepo{user: user}, &httpSessionRepo{}, nil, DefaultArgon2Params)
	service.SetAPITokenStore(store)
	return service, store
}

func TestIssueAndValidateAPIToken(t *testing.T) {
	user := &User{ID: "u1", Email: "staff@example.test", Role: RoleEditor, Active: true}
	service, store := newTokenTestService(t, user)

	plaintext, token, err := service.IssueAPIToken(context.Background(), user.ID, "laptop", APITokenPersonal, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(plaintext, APITokenPrefix) || token.Prefix != plaintext[len(APITokenPrefix):len(APITokenPrefix)+8] {
		t.Fatalf("plaintext=%q prefix=%q", plaintext, token.Prefix)
	}
	if store.tokens[token.ID].TokenHash != HashAPIToken(plaintext) || store.tokens[token.ID].TokenHash == plaintext {
		t.Fatal("token must be stored hashed")
	}
	if got := token.Scopes; len(got) != 1 || got[0] != ScopeAdmin {
		t.Fatalf("personal scopes = %v", got)
	}

	validatedUser, validated, err := service.ValidateAPIToken(context.Background(), plaintext)
	if err != nil || validatedUser.ID != user.ID || validated.ID != token.ID {
		t.Fatalf("validate: user=%v token=%v err=%v", validatedUser, validated, err)
	}
	if store.tokens[token.ID].LastUsedAt == nil {
		t.Fatal("last_used_at not touched")
	}

	if _, _, err := service.ValidateAPIToken(context.Background(), APITokenPrefix+"not-a-real-token-value"); err == nil {
		t.Fatal("unknown token validated")
	}
	if err := service.RevokeAPIToken(context.Background(), token.ID, "someone-else"); err == nil {
		t.Fatal("foreign owner revoked token")
	}
	if err := service.RevokeAPIToken(context.Background(), token.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.ValidateAPIToken(context.Background(), plaintext); err == nil {
		t.Fatal("revoked token validated")
	}
}

func TestAPITokenExpiryAndScopes(t *testing.T) {
	user := &User{ID: "u1", Role: RoleAdmin, Active: true}
	service, _ := newTokenTestService(t, user)

	past := time.Now().UTC().Add(-time.Minute)
	if _, _, err := service.IssueAPIToken(context.Background(), user.ID, "old", APITokenPersonal, nil, &past); err == nil {
		t.Fatal("token issued with past expiry")
	}
	soon := time.Now().UTC().Add(50 * time.Millisecond)
	plaintext, _, err := service.IssueAPIToken(context.Background(), user.ID, "short", APITokenPersonal, nil, &soon)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	if _, _, err := service.ValidateAPIToken(context.Background(), plaintext); err == nil {
		t.Fatal("expired token validated")
	}

	if _, _, err := service.IssueAPIToken(context.Background(), user.ID, "key", APITokenKey, []string{ScopeAdmin}, nil); err == nil {
		t.Fatal("api key issued with admin scope")
	}
	if _, key, err := service.IssueAPIToken(context.Background(), user.ID, "key", APITokenKey, []string{ScopeRead}, nil); err != nil || !key.HasScope(ScopeRead) {
		t.Fatalf("read key: %v %v", key, err)
	}
}

func TestMiddlewareAuthenticatesTokensAndIgnoresCookies(t *testing.T) {
	user := &User{ID: "u1", Email: "staff@example.test", Role: RoleAdmin, Active: true}
	service, _ := newTokenTestService(t, user)
	personal, _, err := service.IssueAPIToken(context.Background(), user.ID, "cli", APITokenPersonal, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	readKey, _, err := service.IssueAPIToken(context.Background(), user.ID, "site", APITokenKey, []string{ScopeRead}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rawSession, _, err := service.CreateSession(context.Background(), user.ID, "ua", "127.0.0.1", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	var seen Principal
	var seenUser *User
	handler := Middleware(service, false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = PrincipalFromContext(r.Context())
		seenUser = UserFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	do := func(method string, configure func(*http.Request)) (*httptest.ResponseRecorder, Principal, *User) {
		seen, seenUser = Principal{}, nil
		req := httptest.NewRequest(method, "/api/v1/admin/me", nil)
		configure(req)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec, seen, seenUser
	}

	rec, principal, u := do(http.MethodGet, func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+personal) })
	if rec.Code != http.StatusNoContent || u == nil || principal.Kind != PrincipalToken || !principal.Allows(http.MethodPost) {
		t.Fatalf("bearer: code=%d user=%v principal=%+v", rec.Code, u, principal)
	}
	rec, principal, u = do(http.MethodGet, func(r *http.Request) { r.Header.Set("X-API-Key", readKey) })
	if rec.Code != http.StatusNoContent || u == nil || principal.Kind != PrincipalAPIKey || !principal.ReadOnly() {
		t.Fatalf("api key: code=%d user=%v principal=%+v", rec.Code, u, principal)
	}
	rec, _, _ = do(http.MethodPost, func(r *http.Request) { r.Header.Set("X-API-Key", readKey) })
	if rec.Code != http.StatusForbidden {
		t.Fatalf("read key POST status = %d", rec.Code)
	}
	var envelope struct{ Code string }
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil || envelope.Code != "forbidden" {
		t.Fatalf("read key POST body = %q", rec.Body.String())
	}

	rec, principal, u = do(http.MethodGet, func(r *http.Request) { r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: rawSession}) })
	if rec.Code != http.StatusNoContent || u == nil || principal.Kind != PrincipalSession {
		t.Fatalf("cookie: code=%d user=%v principal=%+v", rec.Code, u, principal)
	}
	// An invalid token header must not fall back to the (valid) cookie.
	rec, _, u = do(http.MethodGet, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+APITokenPrefix+"invalid-token-value-here")
		r.AddCookie(&http.Cookie{Name: SessionCookieName, Value: rawSession})
	})
	if rec.Code != http.StatusNoContent || u != nil {
		t.Fatalf("invalid token with cookie: code=%d user=%v", rec.Code, u)
	}
}

func TestAdminMeAndTokenEndpoints(t *testing.T) {
	user := &User{ID: "u1", Email: "admin@example.test", Role: RoleAdmin, Active: true}
	service, _ := newTokenTestService(t, user)
	audit := &httpAudit{}
	handler := HTTPHandler{Service: service, Audit: audit}
	mux := http.NewServeMux()
	for _, route := range handler.Routes() {
		if strings.Contains(route.Pattern, "/api/v1/admin/") {
			mux.Handle(route.Pattern, route.Handler)
		}
	}
	asSession := func(req *http.Request) *http.Request {
		ctx := WithUser(req.Context(), user)
		ctx = WithPrincipal(ctx, Principal{Kind: PrincipalSession})
		return req.WithContext(ctx)
	}

	me := httptest.NewRecorder()
	mux.ServeHTTP(me, asSession(httptest.NewRequest(http.MethodGet, "/api/v1/admin/me", nil)))
	if me.Code != http.StatusOK {
		t.Fatalf("me status = %d body=%s", me.Code, me.Body.String())
	}
	var meBody struct {
		Data struct {
			User struct {
				ID    string `json:"id"`
				Email string `json:"email"`
				Role  string `json:"role"`
			} `json:"user"`
			Permissions       []string `json:"permissions"`
			PermissionCatalog []string `json:"permission_catalog"`
			Principal         struct {
				Kind string `json:"kind"`
			} `json:"principal"`
		} `json:"data"`
	}
	if err := json.Unmarshal(me.Body.Bytes(), &meBody); err != nil {
		t.Fatal(err)
	}
	if meBody.Data.User.ID != "u1" || meBody.Data.User.Role != "admin" || meBody.Data.Principal.Kind != "session" || len(meBody.Data.PermissionCatalog) == 0 || len(meBody.Data.Permissions) != len(meBody.Data.PermissionCatalog) {
		t.Fatalf("me body = %s", me.Body.String())
	}

	created := httptest.NewRecorder()
	req := asSession(httptest.NewRequest(http.MethodPost, "/api/v1/admin/account/tokens", strings.NewReader(`{"name":"cli","expires_in_days":30}`)))
	mux.ServeHTTP(created, req)
	if created.Code != http.StatusCreated {
		t.Fatalf("issue status = %d body=%s", created.Code, created.Body.String())
	}
	var issued struct {
		Data struct {
			Token string `json:"token"`
			Item  struct {
				ID        string  `json:"id"`
				ExpiresAt *string `json:"expires_at"`
			} `json:"item"`
		} `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &issued); err != nil || !IsAPIToken(issued.Data.Token) || issued.Data.Item.ExpiresAt == nil {
		t.Fatalf("issue body = %s err=%v", created.Body.String(), err)
	}

	invalid := httptest.NewRecorder()
	mux.ServeHTTP(invalid, asSession(httptest.NewRequest(http.MethodPost, "/api/v1/admin/account/tokens", strings.NewReader(`{"name":""}`))))
	if invalid.Code != http.StatusUnprocessableEntity || !strings.Contains(invalid.Body.String(), `"name":"required"`) {
		t.Fatalf("validation = %d %s", invalid.Code, invalid.Body.String())
	}

	// A token principal may not mint further tokens.
	fromToken := httptest.NewRequest(http.MethodPost, "/api/v1/admin/account/tokens", strings.NewReader(`{"name":"x"}`))
	fromToken = fromToken.WithContext(WithPrincipal(WithUser(fromToken.Context(), user), Principal{Kind: PrincipalToken}))
	denied := httptest.NewRecorder()
	mux.ServeHTTP(denied, fromToken)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("token-issued-by-token status = %d", denied.Code)
	}

	listed := httptest.NewRecorder()
	mux.ServeHTTP(listed, asSession(httptest.NewRequest(http.MethodGet, "/api/v1/admin/account/tokens", nil)))
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), issued.Data.Item.ID) || strings.Contains(listed.Body.String(), issued.Data.Token) {
		t.Fatalf("list = %d %s", listed.Code, listed.Body.String())
	}

	key := httptest.NewRecorder()
	mux.ServeHTTP(key, asSession(httptest.NewRequest(http.MethodPost, "/api/v1/admin/api-keys", strings.NewReader(`{"name":"site","scope":"read"}`))))
	if key.Code != http.StatusCreated {
		t.Fatalf("api key status = %d body=%s", key.Code, key.Body.String())
	}
	badScope := httptest.NewRecorder()
	mux.ServeHTTP(badScope, asSession(httptest.NewRequest(http.MethodPost, "/api/v1/admin/api-keys", strings.NewReader(`{"name":"site","scope":"admin"}`))))
	if badScope.Code != http.StatusUnprocessableEntity {
		t.Fatalf("admin-scope key status = %d", badScope.Code)
	}

	revoked := httptest.NewRecorder()
	mux.ServeHTTP(revoked, asSession(httptest.NewRequest(http.MethodDelete, "/api/v1/admin/account/tokens/"+issued.Data.Item.ID, nil)))
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d", revoked.Code)
	}
	if _, _, err := service.ValidateAPIToken(context.Background(), issued.Data.Token); err == nil {
		t.Fatal("revoked token still validates")
	}
	missing := httptest.NewRecorder()
	mux.ServeHTTP(missing, asSession(httptest.NewRequest(http.MethodDelete, "/api/v1/admin/account/tokens/nope", nil)))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("revoke missing status = %d", missing.Code)
	}

	editor := &User{ID: "u2", Role: RoleEditor, Active: true}
	keysAsEditor := httptest.NewRequest(http.MethodGet, "/api/v1/admin/api-keys", nil)
	keysAsEditor = keysAsEditor.WithContext(WithPrincipal(WithUser(keysAsEditor.Context(), editor), Principal{Kind: PrincipalSession}))
	forbidden := httptest.NewRecorder()
	mux.ServeHTTP(forbidden, keysAsEditor)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("editor api-keys status = %d", forbidden.Code)
	}
}

func TestCSRFSkippedForAPITokenRequests(t *testing.T) {
	secured := security.CSRFMiddleware(false, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))

	bearer := httptest.NewRequest(http.MethodPost, "/api/v1/admin/content/faq", strings.NewReader(`{}`))
	bearer.Header.Set("Authorization", "Bearer "+APITokenPrefix+"abcdefghijklmnop")
	rec := httptest.NewRecorder()
	secured.ServeHTTP(rec, bearer)
	if rec.Code != http.StatusNoContent || rec.Header().Get("Set-Cookie") != "" {
		t.Fatalf("bearer: status=%d set-cookie=%q", rec.Code, rec.Header().Get("Set-Cookie"))
	}

	key := httptest.NewRequest(http.MethodPost, "/api/v1/admin/content/faq", strings.NewReader(`{}`))
	key.Header.Set("X-API-Key", APITokenPrefix+"abcdefghijklmnop")
	rec = httptest.NewRecorder()
	secured.ServeHTTP(rec, key)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("api key: status=%d", rec.Code)
	}

	cookieOnly := httptest.NewRequest(http.MethodPost, "/api/v1/admin/content/faq", strings.NewReader(`{}`))
	cookieOnly.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "token"})
	rec = httptest.NewRecorder()
	secured.ServeHTTP(rec, cookieOnly)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cookie without header: status=%d", rec.Code)
	}
	withHeader := httptest.NewRequest(http.MethodPost, "/api/v1/admin/content/faq", strings.NewReader(`{}`))
	withHeader.AddCookie(&http.Cookie{Name: security.CSRFCookieName, Value: "token"})
	withHeader.Header.Set(security.CSRFHeaderName, "token")
	rec = httptest.NewRecorder()
	secured.ServeHTTP(rec, withHeader)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("cookie with header: status=%d", rec.Code)
	}
}
