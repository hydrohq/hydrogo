package auth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type OAuthProviderConfig struct {
	Name, ClientID, ClientSecret            string
	AuthorizationURL, TokenURL, UserInfoURL string
	RedirectURL                             string
	JWKSURL                                 string
	Scopes                                  []string
	Apple                                   bool
}

type OAuthIdentity struct {
	Provider, Subject, Email, Name, Picture string
	EmailVerified                           bool
}

// OAuthAccountResolver must map a verified provider subject to an existing
// application identity. It owns account linking and provisioning policy.
type OAuthAccountResolver interface {
	ResolveOAuth(context.Context, OAuthIdentity) (*User, error)
}

type OAuthIdentityResolverFunc func(context.Context, OAuthIdentity) (*User, error)

func (f OAuthIdentityResolverFunc) ResolveOAuth(ctx context.Context, identity OAuthIdentity) (*User, error) {
	return f(ctx, identity)
}

// OAuthFlow is the provider-facing half of a social sign-in: it validates
// the provider name, runs the PKCE + signed-state dance, exchanges the
// authorization code and verifies the returned identity (including Apple's
// signed id_token). It knows nothing about accounts, roles or sessions —
// callers (SocialOAuthHandler here for admin staff, and the member-facing
// handler in modules/members/transport) each supply their own resolver and
// session issuance on top of the OAuthIdentity it returns. This split exists
// so the token-exchange and Apple JWT verification code, the parts most
// worth getting right once, are never duplicated between the two.
type OAuthFlow struct {
	Providers              map[string]OAuthProviderConfig
	HTTPClient             *http.Client
	StateKey               []byte
	Secure                 bool
	AllowInsecureLocalhost bool
	Now                    func() time.Time
	// CookieName, CookiePath, DefaultReturnTo and CancelRedirect default to
	// the admin surface's historical values ("hydro_oauth_state",
	// "/auth/oauth/", "/admin", "/login?social=cancelled") when empty, so
	// existing admin wiring and tests are unaffected. A second consumer
	// (member sign-in) sets its own values so the two flows' state cookies
	// and redirects never collide, even though both can run in the same
	// process.
	CookieName      string
	CookiePath      string
	DefaultReturnTo string
	CancelRedirect  string
}

type SocialOAuthHandler struct {
	OAuthFlow
	Resolver OAuthAccountResolver
	Service  *AuthService
	Audit    AuditRecorder
}

type oauthState struct {
	State, Verifier, Nonce, ReturnTo string
	Expires                          int64
}
type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
}

func DefaultOAuthProvider(name, clientID, clientSecret, redirectURL string) (OAuthProviderConfig, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "google":
		return OAuthProviderConfig{Name: "google", ClientID: clientID, ClientSecret: clientSecret, RedirectURL: redirectURL, AuthorizationURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token", UserInfoURL: "https://openidconnect.googleapis.com/v1/userinfo", Scopes: []string{"openid", "email", "profile"}}, nil // #nosec G101 -- provider endpoint URLs, not credentials
	case "github":
		return OAuthProviderConfig{Name: "github", ClientID: clientID, ClientSecret: clientSecret, RedirectURL: redirectURL, AuthorizationURL: "https://github.com/login/oauth/authorize", TokenURL: "https://github.com/login/oauth/access_token", UserInfoURL: "https://api.github.com/user", Scopes: []string{"read:user", "user:email"}}, nil // #nosec G101 -- provider endpoint URLs, not credentials
	case "apple":
		return OAuthProviderConfig{Name: "apple", ClientID: clientID, ClientSecret: clientSecret, RedirectURL: redirectURL, AuthorizationURL: "https://appleid.apple.com/auth/authorize", TokenURL: "https://appleid.apple.com/auth/token", JWKSURL: "https://appleid.apple.com/auth/keys", Scopes: []string{"openid", "email", "name"}, Apple: true}, nil // #nosec G101 -- provider endpoint URLs, not credentials
	default:
		return OAuthProviderConfig{}, errors.New("unsupported social provider")
	}
}

func (h SocialOAuthHandler) Routes() []HTTPRoute {
	if len(h.Providers) == 0 || h.Resolver == nil || h.Service == nil || len(h.StateKey) < 32 {
		return nil
	}
	return []HTTPRoute{
		{Pattern: "GET /auth/oauth/{provider}/start", Public: true, Handler: http.HandlerFunc(h.Start)},
		{Pattern: "GET /auth/oauth/{provider}/callback", Public: true, Handler: http.HandlerFunc(h.callback)},
	}
}

// callback resolves the generic exchange, then applies admin policy: only an
// existing, active admin or editor account may be signed in this way — the
// resolver (OAuthIdentityStore) never creates one.
func (h SocialOAuthHandler) callback(w http.ResponseWriter, r *http.Request) {
	identity, returnTo, handled := h.HandleCallback(w, r)
	if handled {
		return
	}
	user, err := h.Resolver.ResolveOAuth(r.Context(), identity)
	if err != nil || user == nil || !user.Active {
		http.Error(w, "social login could not be completed", http.StatusUnauthorized)
		return
	}
	if user.Role != RoleAdmin && user.Role != RoleEditor {
		http.Error(w, "social login could not be completed", http.StatusUnauthorized)
		return
	}
	token, session, err := h.Service.CreateSession(r.Context(), user.ID, r.UserAgent(), clientIP(r), DefaultSessionTTL)
	if err != nil {
		http.Error(w, "social login could not be completed", http.StatusInternalServerError)
		return
	}
	SetSessionCookie(w, token, session.ExpiresAt, h.Secure)
	if h.Audit != nil {
		_ = h.Audit.Record(r.Context(), "auth.oauth.login", user.ID)
	}
	http.Redirect(w, r, returnTo, http.StatusSeeOther) // #nosec G710 -- returnTo was cleaned by safeOAuthReturnTo and travels in the HMAC-signed state cookie
}

// Start begins a provider's authorization-code + PKCE flow: it validates the
// provider and its configured URLs, mints a signed, time-boxed state cookie
// (and a PKCE verifier, and a nonce for Apple), and redirects to the
// provider's authorization endpoint.
func (f OAuthFlow) Start(w http.ResponseWriter, r *http.Request) {
	provider, ok := f.provider(r.PathValue("provider"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !validOAuthURL(provider.AuthorizationURL, f.AllowInsecureLocalhost) || !validOAuthURL(provider.RedirectURL, f.AllowInsecureLocalhost) || provider.ClientID == "" {
		http.Error(w, "social login unavailable", http.StatusServiceUnavailable)
		return
	}
	returnTo := safeOAuthReturnTo(r.URL.Query().Get("return_to"), f.defaultReturnTo())
	state, err := randomOAuthValue(32)
	if err != nil {
		http.Error(w, "social login unavailable", http.StatusInternalServerError)
		return
	}
	verifier, err := randomOAuthValue(48)
	if err != nil {
		http.Error(w, "social login unavailable", http.StatusInternalServerError)
		return
	}
	nonce, err := randomOAuthValue(32)
	if err != nil {
		http.Error(w, "social login unavailable", http.StatusInternalServerError)
		return
	}
	now := f.now()
	serialized, _ := json.Marshal(oauthState{State: state, Verifier: verifier, Nonce: nonce, ReturnTo: returnTo, Expires: now.Add(10 * time.Minute).Unix()})
	encoded := base64.RawURLEncoding.EncodeToString(serialized)
	mac := hmac.New(sha256.New, f.StateKey)
	_, _ = mac.Write([]byte(encoded))
	value := encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	http.SetCookie(w, &http.Cookie{Name: f.cookieName(), Value: value, Path: f.cookiePath(), MaxAge: 600, HttpOnly: true, Secure: f.Secure, SameSite: http.SameSiteLaxMode}) // #nosec G124 -- Secure follows the secure_cookies setting (off only for plain-HTTP development); the cookie is HttpOnly and SameSite
	u, _ := url.Parse(provider.AuthorizationURL)
	query := u.Query()
	query.Set("response_type", "code")
	query.Set("client_id", provider.ClientID)
	query.Set("redirect_uri", provider.RedirectURL)
	query.Set("scope", strings.Join(provider.Scopes, " "))
	query.Set("state", state)
	query.Set("code_challenge_method", "S256")
	challenge := sha256.Sum256([]byte(verifier))
	query.Set("code_challenge", base64.RawURLEncoding.EncodeToString(challenge[:]))
	if provider.Apple {
		query.Set("nonce", nonce)
		query.Set("response_mode", "query")
	}
	u.RawQuery = query.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

// HandleCallback is the provider-facing half of a callback: it always clears
// the state cookie, handles a user-cancelled flow, validates and decodes the
// signed state, and exchanges the code for a verified identity. On any
// outcome that the caller should not continue past (cancellation, a missing
// or invalid state, or a failed exchange) it writes the response itself and
// returns handled=true; the caller's own account-resolution and session
// logic only runs when handled is false.
func (f OAuthFlow) HandleCallback(w http.ResponseWriter, r *http.Request) (identity OAuthIdentity, returnTo string, handled bool) {
	provider, ok := f.provider(r.PathValue("provider"))
	if !ok {
		http.NotFound(w, r)
		return OAuthIdentity{}, "", true
	}
	if r.URL.Query().Get("error") != "" {
		clearOAuthCookie(w, f.Secure, f.cookieName(), f.cookiePath())
		http.Redirect(w, r, f.cancelRedirect(), http.StatusSeeOther)
		return OAuthIdentity{}, "", true
	}
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	cookie, err := r.Cookie(f.cookieName())
	if err != nil || state == "" || code == "" {
		http.Error(w, "invalid social login response", http.StatusBadRequest)
		return OAuthIdentity{}, "", true
	}
	clearOAuthCookie(w, f.Secure, f.cookieName(), f.cookiePath())
	pending, err := f.decodeState(cookie.Value)
	if err != nil || subtle.ConstantTimeCompare([]byte(pending.State), []byte(state)) != 1 || pending.Expires < f.now().Unix() {
		http.Error(w, "social login expired; please try again", http.StatusBadRequest)
		return OAuthIdentity{}, "", true
	}
	identity, err = f.exchange(r.Context(), provider, code, pending)
	if err != nil {
		http.Error(w, "social login could not be completed", http.StatusUnauthorized)
		return OAuthIdentity{}, "", true
	}
	return identity, pending.ReturnTo, false
}

func (f OAuthFlow) cookieName() string {
	if f.CookieName != "" {
		return f.CookieName
	}
	return "hydro_oauth_state"
}

func (f OAuthFlow) cookiePath() string {
	if f.CookiePath != "" {
		return f.CookiePath
	}
	return "/auth/oauth/"
}

func (f OAuthFlow) defaultReturnTo() string {
	if f.DefaultReturnTo != "" {
		return f.DefaultReturnTo
	}
	return "/admin"
}

func (f OAuthFlow) cancelRedirect() string {
	if f.CancelRedirect != "" {
		return f.CancelRedirect
	}
	return "/login?social=cancelled"
}

func (f OAuthFlow) exchange(ctx context.Context, provider OAuthProviderConfig, code string, pending oauthState) (OAuthIdentity, error) {
	if !validOAuthURL(provider.TokenURL, f.AllowInsecureLocalhost) || (!provider.Apple && !validOAuthURL(provider.UserInfoURL, f.AllowInsecureLocalhost)) {
		return OAuthIdentity{}, errors.New("invalid provider endpoint")
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "client_id": {provider.ClientID}, "redirect_uri": {provider.RedirectURL}, "code_verifier": {pending.Verifier}}
	if provider.ClientSecret != "" {
		form.Set("client_secret", provider.ClientSecret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return OAuthIdentity{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	client := f.client()
	response, err := client.Do(req)
	if err != nil {
		return OAuthIdentity{}, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20+1))
	if err != nil || len(data) > 1<<20 || response.StatusCode != http.StatusOK {
		return OAuthIdentity{}, errors.New("provider token exchange failed")
	}
	var token oauthTokenResponse
	if err := json.Unmarshal(data, &token); err != nil {
		return OAuthIdentity{}, err
	}
	if provider.Apple {
		return verifyAppleIdentity(ctx, token.IDToken, provider.ClientID, pending.Nonce, provider.JWKSURL, client, f.AllowInsecureLocalhost)
	}
	if token.AccessToken == "" {
		return OAuthIdentity{}, errors.New("provider returned no access token")
	}
	identity, err := f.fetchIdentity(ctx, provider, token.AccessToken, client)
	if err != nil {
		return OAuthIdentity{}, err
	}
	identity.Provider = provider.Name
	if identity.Subject == "" || identity.Email == "" || !identity.EmailVerified {
		return OAuthIdentity{}, errors.New("provider identity is incomplete or unverified")
	}
	return identity, nil
}

func (f OAuthFlow) fetchIdentity(ctx context.Context, provider OAuthProviderConfig, accessToken string, client *http.Client) (OAuthIdentity, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.UserInfoURL, nil)
	if err != nil {
		return OAuthIdentity{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "HydroGo")
	resp, err := client.Do(req)
	if err != nil {
		return OAuthIdentity{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil || len(data) > 1<<20 || resp.StatusCode != http.StatusOK {
		return OAuthIdentity{}, errors.New("provider profile lookup failed")
	}
	var raw struct {
		ID       json.RawMessage `json:"id"`
		Sub      string          `json:"sub"`
		Email    string          `json:"email"`
		Verified bool            `json:"email_verified"`
		Name     string          `json:"name"`
		Login    string          `json:"login"`
		Avatar   string          `json:"avatar_url"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return OAuthIdentity{}, err
	}
	subject := raw.Sub
	if subject == "" {
		subject = strings.Trim(string(raw.ID), "\"")
	}
	name := raw.Name
	if name == "" {
		name = raw.Login
	}
	identity := OAuthIdentity{Provider: provider.Name, Subject: subject, Email: strings.ToLower(strings.TrimSpace(raw.Email)), EmailVerified: raw.Verified, Name: name, Picture: raw.Avatar}
	if provider.Name == "github" {
		return f.fetchGitHubEmail(ctx, provider, accessToken, identity, client)
	}
	return identity, nil
}

func (f OAuthFlow) fetchGitHubEmail(ctx context.Context, provider OAuthProviderConfig, accessToken string, identity OAuthIdentity, client *http.Client) (OAuthIdentity, error) {
	u, err := url.Parse(provider.UserInfoURL)
	if err != nil {
		return identity, err
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	if strings.HasSuffix(u.Path, "/user") {
		u.Path += "/emails"
	} else {
		u.Path += "/user/emails"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return identity, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "HydroGo")
	resp, err := client.Do(req)
	if err != nil {
		return identity, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20+1))
	if err != nil || len(data) > 1<<20 || resp.StatusCode != http.StatusOK {
		return identity, errors.New("provider email lookup failed")
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(data, &emails); err != nil {
		return identity, err
	}
	for _, item := range emails {
		if item.Primary && item.Verified {
			identity.Email = strings.ToLower(strings.TrimSpace(item.Email))
			identity.EmailVerified = true
			break
		}
	}
	if identity.Email == "" {
		return identity, errors.New("provider has no verified email")
	}
	return identity, nil
}

func (f OAuthFlow) decodeState(value string) (oauthState, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return oauthState{}, errors.New("invalid state cookie")
	}
	mac := hmac.New(sha256.New, f.StateKey)
	_, _ = mac.Write([]byte(parts[0]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		return oauthState{}, errors.New("invalid state signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return oauthState{}, err
	}
	var state oauthState
	if err := json.Unmarshal(payload, &state); err != nil {
		return state, err
	}
	return state, nil
}

func (f OAuthFlow) provider(name string) (OAuthProviderConfig, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	p, ok := f.Providers[name]
	if !ok {
		return OAuthProviderConfig{}, false
	}
	p.Name = strings.ToLower(p.Name)
	if p.Name == "" {
		p.Name = name
	}
	return p, p.Name == name && (p.Name == "google" || p.Name == "github" || p.Name == "apple")
}
func (f OAuthFlow) client() *http.Client {
	client := f.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	copy := *client
	copy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &copy
}
func (f OAuthFlow) now() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now()
}
func clearOAuthCookie(w http.ResponseWriter, secure bool, name, path string) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: path, MaxAge: -1, HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode}) // #nosec G124 -- Secure follows the secure_cookies setting (off only for plain-HTTP development); the cookie is HttpOnly and SameSite
}
func randomOAuthValue(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
func safeOAuthReturnTo(value, fallback string) string {
	if value == "" {
		return fallback
	}
	u, err := url.Parse(value)
	if err != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") || strings.ContainsAny(value, "\r\n\\") {
		return fallback
	}
	return value
}
func validOAuthURL(value string, allowLocal bool) bool {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || u.User != nil {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	return allowLocal && u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1")
}

func verifyAppleIdentity(ctx context.Context, token, audience, nonce, jwksURL string, client *http.Client, allowLocal bool) (OAuthIdentity, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return OAuthIdentity{}, errors.New("invalid identity token")
	}
	decode := func(value string) ([]byte, error) { return base64.RawURLEncoding.DecodeString(value) }
	headerData, err := decode(parts[0])
	if err != nil {
		return OAuthIdentity{}, err
	}
	var header struct {
		Algorithm string `json:"alg"`
		KeyID     string `json:"kid"`
	}
	if json.Unmarshal(headerData, &header) != nil || header.KeyID == "" || (header.Algorithm != "RS256" && header.Algorithm != "ES256") {
		return OAuthIdentity{}, errors.New("unsupported identity token signature")
	}
	if jwksURL == "" {
		jwksURL = "https://appleid.apple.com/auth/keys"
	}
	if !validOAuthURL(jwksURL, allowLocal) {
		return OAuthIdentity{}, errors.New("invalid identity key endpoint")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return OAuthIdentity{}, err
	}
	response, err := client.Do(req)
	if err != nil {
		return OAuthIdentity{}, err
	}
	defer response.Body.Close()
	keyData, err := io.ReadAll(io.LimitReader(response.Body, 1<<20+1))
	if err != nil || len(keyData) > 1<<20 || response.StatusCode != http.StatusOK {
		return OAuthIdentity{}, errors.New("identity keys unavailable")
	}
	var keys struct {
		Keys []struct{ Kty, Kid, N, E, Crv, X, Y string } `json:"keys"`
	}
	if json.Unmarshal(keyData, &keys) != nil {
		return OAuthIdentity{}, errors.New("invalid identity keys")
	}
	var selected *struct{ Kty, Kid, N, E, Crv, X, Y string }
	for i := range keys.Keys {
		if keys.Keys[i].Kid == header.KeyID {
			selected = &keys.Keys[i]
			break
		}
	}
	if selected == nil {
		return OAuthIdentity{}, errors.New("identity signing key not found")
	}
	signed := []byte(parts[0] + "." + parts[1])
	signature, err := decode(parts[2])
	if err != nil {
		return OAuthIdentity{}, err
	}
	digest := sha256.Sum256(signed)
	switch header.Algorithm {
	case "RS256":
		if selected.Kty != "RSA" {
			return OAuthIdentity{}, errors.New("identity key type mismatch")
		}
		n, e1 := decode(selected.N)
		e, e2 := decode(selected.E)
		if e1 != nil || e2 != nil || len(e) == 0 {
			return OAuthIdentity{}, errors.New("invalid RSA identity key")
		}
		exponent := 0
		for _, b := range e {
			exponent = exponent<<8 | int(b)
		}
		if exponent < 3 {
			return OAuthIdentity{}, errors.New("invalid RSA exponent")
		}
		pub := &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: exponent}
		if pub.N.BitLen() < 2048 || rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], signature) != nil {
			return OAuthIdentity{}, errors.New("invalid identity token signature")
		}
	case "ES256":
		if selected.Kty != "EC" || selected.Crv != "P-256" || len(signature) != 64 {
			return OAuthIdentity{}, errors.New("invalid EC identity key")
		}
		x, e1 := decode(selected.X)
		y, e2 := decode(selected.Y)
		if e1 != nil || e2 != nil {
			return OAuthIdentity{}, errors.New("invalid EC identity key")
		}
		if !ecdsa.Verify(&ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(x), Y: new(big.Int).SetBytes(y)}, digest[:], new(big.Int).SetBytes(signature[:32]), new(big.Int).SetBytes(signature[32:])) {
			return OAuthIdentity{}, errors.New("invalid identity token signature")
		}
	}
	payload, err := decode(parts[1])
	if err != nil {
		return OAuthIdentity{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.UseNumber()
	var claims struct {
		Issuer        string          `json:"iss"`
		Subject       string          `json:"sub"`
		Audience      json.RawMessage `json:"aud"`
		Expires       json.Number     `json:"exp"`
		IssuedAt      json.Number     `json:"iat"`
		Nonce         string          `json:"nonce"`
		Email         string          `json:"email"`
		EmailVerified any             `json:"email_verified"`
	}
	if decoder.Decode(&claims) != nil {
		return OAuthIdentity{}, errors.New("invalid identity claims")
	}
	var aud string
	if json.Unmarshal(claims.Audience, &aud) != nil {
		var values []string
		_ = json.Unmarshal(claims.Audience, &values)
		for _, v := range values {
			if v == audience {
				aud = v
				break
			}
		}
	}
	exp, _ := strconv.ParseInt(string(claims.Expires), 10, 64)
	iat, _ := strconv.ParseInt(string(claims.IssuedAt), 10, 64)
	now := time.Now().Unix()
	if claims.Issuer != "https://appleid.apple.com" || aud != audience || claims.Subject == "" || exp <= now || iat < now-600 || iat > now+300 || subtle.ConstantTimeCompare([]byte(claims.Nonce), []byte(nonce)) != 1 {
		return OAuthIdentity{}, errors.New("identity token claims rejected")
	}
	verified := false
	switch value := claims.EmailVerified.(type) {
	case bool:
		verified = value
	case string:
		verified = value == "true"
	}
	if !verified || strings.TrimSpace(claims.Email) == "" {
		return OAuthIdentity{}, errors.New("verified email claim required")
	}
	return OAuthIdentity{Provider: "apple", Subject: claims.Subject, Email: strings.ToLower(strings.TrimSpace(claims.Email)), EmailVerified: true}, nil
}
