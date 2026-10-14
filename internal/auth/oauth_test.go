package auth

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/testhttp"
)

type oauthSessions struct{ created *Session }

func (s *oauthSessions) Create(_ context.Context, session *Session) error {
	s.created = session
	return nil
}
func (*oauthSessions) GetByTokenHash(context.Context, string) (*Session, error) {
	return nil, ErrSessionNotFound
}
func (*oauthSessions) DeleteByTokenHash(context.Context, string) error         { return nil }
func (*oauthSessions) DeleteByUserID(context.Context, string) error            { return nil }
func (*oauthSessions) DeleteExpired(context.Context, time.Time) (int64, error) { return 0, nil }
func (*oauthSessions) Touch(context.Context, string, time.Time) error          { return nil }

func TestSocialOAuthStartAndCallbackUseStatePKCEAndCreateOpaqueSession(t *testing.T) {
	var verifier string
	const upstream = "http://127.0.0.1:19091"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19091": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			verifier = r.Form.Get("code_verifier")
			if verifier == "" || r.Form.Get("client_secret") != "secret" {
				t.Errorf("missing PKCE/client credentials: %v", r.Form)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"upstream-token"}`))
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer upstream-token" {
				t.Errorf("auth header=%q", r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"sub":"google-subject","email":"admin@example.test","email_verified":true,"name":"Admin"}`))
		default:
			http.NotFound(w, r)
		}
	})})
	sessions := &oauthSessions{}
	service := &AuthService{sessionRepo: sessions}
	provider := OAuthProviderConfig{Name: "google", ClientID: "client", ClientSecret: "secret", AuthorizationURL: upstream + "/authorize", TokenURL: upstream + "/token", UserInfoURL: upstream + "/userinfo", RedirectURL: upstream + "/callback", Scopes: []string{"openid", "email"}}
	handler := SocialOAuthHandler{Providers: map[string]OAuthProviderConfig{"google": provider}, HTTPClient: client, Resolver: OAuthIdentityResolverFunc(func(_ context.Context, identity OAuthIdentity) (*User, error) {
		if identity.Subject != "google-subject" || !identity.EmailVerified {
			t.Fatalf("identity=%+v", identity)
		}
		return &User{ID: "admin-1", Role: RoleAdmin, Active: true}, nil
	}), Service: service, StateKey: []byte(strings.Repeat("s", 32)), AllowInsecureLocalhost: true, Now: func() time.Time { return time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC) }}
	start := httptest.NewRecorder()
	startRequest := httptest.NewRequest(http.MethodGet, "/auth/oauth/google/start?return_to=%2Fadmin%2Fpages", nil)
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
	challenge := authorize.Query().Get("code_challenge")
	cookie := start.Result().Cookies()[0]
	pending, err := handler.decodeState(cookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(pending.Verifier))
	if pending.State != state || challenge != base64.RawURLEncoding.EncodeToString(sum[:]) || pending.ReturnTo != "/admin/pages" {
		t.Fatalf("state/PKCE mismatch: %+v", pending)
	}
	callback := httptest.NewRequest(http.MethodGet, "/auth/oauth/google/callback?state="+url.QueryEscape(state)+"&code=once", nil)
	callback.SetPathValue("provider", "google")
	callback.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.callback(response, callback)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/admin/pages" || sessions.created == nil || sessions.created.UserID != "admin-1" || verifier != pending.Verifier {
		t.Fatalf("callback status=%d location=%q session=%+v verifier-match=%v", response.Code, response.Header().Get("Location"), sessions.created, verifier == pending.Verifier)
	}
	issuedSession := false
	for _, item := range response.Result().Cookies() {
		if item.Name == SessionCookieName && item.Value != "" {
			issuedSession = true
		}
	}
	if !issuedSession {
		t.Fatal("Hydro session cookie was not issued")
	}
}

func TestSocialOAuthRejectsTamperedStateAndUnverifiedEmail(t *testing.T) {
	key := []byte(strings.Repeat("k", 32))
	state := oauthState{State: "right", Verifier: "verifier", Expires: time.Now().Add(time.Minute).Unix()}
	payload, _ := json.Marshal(state)
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(encoded))
	cookie := encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) + "tampered"
	handler := SocialOAuthHandler{StateKey: key}
	if _, err := handler.decodeState(cookie); err == nil {
		t.Fatal("accepted tampered OAuth state")
	}
	const upstream = "http://127.0.0.1:19092"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19092": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"sub":"id","email":"user@example.test","email_verified":false}`))
	})})
	handler.AllowInsecureLocalhost = true
	identity, err := handler.fetchIdentity(context.Background(), OAuthProviderConfig{Name: "google", UserInfoURL: upstream}, "token", client)
	if err != nil {
		t.Fatal(err)
	}
	if identity.EmailVerified {
		t.Fatal("accepted unverified email")
	}
}

func TestVerifyAppleIdentityTokenChecksSignatureAudienceNonceAndExpiry(t *testing.T) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const upstream = "http://127.0.0.1:19093"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19093": http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		e := bigEndian(private.PublicKey.E)
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"kty": "RSA", "kid": "test-key", "n": base64.RawURLEncoding.EncodeToString(private.PublicKey.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(e)}}})
	})})
	now := time.Now().UTC()
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "kid": "test-key"})
	claims, _ := json.Marshal(map[string]any{"iss": "https://appleid.apple.com", "sub": "apple-sub", "aud": "client", "exp": now.Add(time.Minute).Unix(), "iat": now.Unix(), "nonce": "expected", "email": "person@example.test", "email_verified": true})
	head := base64.RawURLEncoding.EncodeToString(header)
	body := base64.RawURLEncoding.EncodeToString(claims)
	signed := head + "." + body
	digest := sha256.Sum256([]byte(signed))
	signature, err := rsa.SignPKCS1v15(rand.Reader, private, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	token := signed + "." + base64.RawURLEncoding.EncodeToString(signature)
	identity, err := verifyAppleIdentity(context.Background(), token, "client", "expected", upstream, client, true)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Provider != "apple" || identity.Subject != "apple-sub" || !identity.EmailVerified {
		t.Fatalf("identity=%+v", identity)
	}
	if _, err := verifyAppleIdentity(context.Background(), token, "wrong", "expected", upstream, client, true); err == nil {
		t.Fatal("accepted wrong audience")
	}
}

func bigEndian(value int) []byte {
	var out []byte
	for value > 0 {
		out = append([]byte{byte(value)}, out...)
		value >>= 8
	}
	return out
}
