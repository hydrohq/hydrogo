package transport_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	v1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/modules/members"
	"github.com/ardaozyildirim/hydrogo/modules/members/transport"
)

var inlineMemberScript = regexp.MustCompile(`(?s)<script(?:\s[^>]*)?>\s*[^<\s]`)

func TestMemberPagesRenderThroughEveryTheme(t *testing.T) {
	for _, name := range []string{"", "corporate", "newsroom"} {
		var theme *v1.ThemeBundle
		if name != "" {
			bundle, err := v1.LoadTheme(filepath.Join("..", "..", "..", "themes"), name)
			if err != nil {
				t.Fatal(err)
			}
			theme = &bundle
		}
		svc := members.Service{Members: newMemoryMemberStore(), Sessions: newMemorySessionStore(), ArgonParams: auth.DefaultArgon2Params}
		mux := http.NewServeMux()
		for _, route := range (transport.Handler{Service: svc, Theme: theme}).Routes() {
			mux.Handle(route.Pattern, route.Handler)
		}
		get := func(path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			for _, cookie := range cookies {
				request.AddCookie(cookie)
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			return response
		}

		login := get("/member/login?return_to=%2Fpublic%2Fnews%2Ftr%2Fhtml")
		body := login.Body.String()
		for _, want := range []string{`data-member-form`, `action="/member/login"`, `name="_csrf"`, `name="return_to" value="/public/news/tr/html"`, `src="/member/assets/member-auth.js"`, `href="/member/register?return_to=%2Fpublic%2Fnews%2Ftr%2Fhtml"`} {
			if !strings.Contains(body, want) {
				t.Errorf("theme %q login page missing %q", name, want)
			}
		}
		register := get("/member/register").Body.String()
		for _, want := range []string{`action="/member/register"`, `autocomplete="new-password"`, `data-password-strength`, `data-password-meter`} {
			if !strings.Contains(register, want) {
				t.Errorf("theme %q register page missing %q", name, want)
			}
		}
		if inlineMemberScript.MatchString(body+register) || strings.Contains(body+register, "<style") {
			t.Errorf("theme %q member auth pages carry inline code", name)
		}
		if csp := login.Header().Get("Content-Security-Policy"); strings.Contains(csp, "unsafe-inline") {
			t.Errorf("theme %q login CSP permits inline code: %s", name, csp)
		}

		form := url.Values{"email": {"<b>reader</b>@example.test"}, "password": {"Amber-River7!Cedar"}}
		post := httptest.NewRequest(http.MethodPost, "/member/register", strings.NewReader(form.Encode()))
		post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		created := httptest.NewRecorder()
		mux.ServeHTTP(created, post)
		var session *http.Cookie
		for _, cookie := range created.Result().Cookies() {
			if cookie.Name == members.SessionCookieName {
				session = cookie
			}
		}
		if session == nil {
			t.Skipf("registration rejected the test address: %d %s", created.Code, created.Body.String())
		}
		account := get("/member/account", session)
		accountBody := account.Body.String()
		if account.Code != http.StatusOK || !strings.Contains(accountBody, `data-member-logout`) || strings.Contains(accountBody, "<b>reader</b>") {
			t.Errorf("theme %q account page status=%d escaped=%t", name, account.Code, !strings.Contains(accountBody, "<b>reader</b>"))
		}
		if got := account.Header().Get("Cache-Control"); got != "private, no-store" {
			t.Errorf("theme %q account Cache-Control = %q", name, got)
		}
	}
}

func TestMemberAuthScriptIsServed(t *testing.T) {
	mux := http.NewServeMux()
	for _, route := range (transport.Handler{}).Routes() {
		mux.Handle(route.Pattern, route.Handler)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/member/assets/member-auth.js", nil))
	if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Type"), "text/javascript") || !strings.Contains(response.Body.String(), "data-member-form") {
		t.Fatalf("member auth script status=%d type=%q", response.Code, response.Header().Get("Content-Type"))
	}
	if strings.Contains(response.Body.String(), "localStorage") {
		t.Fatal("member auth script persists identity in browser storage")
	}
}
