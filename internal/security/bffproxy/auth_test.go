package bffproxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/security/bffsession"
)

func TestAuthHandlerLoginAndLogoutExchange(t *testing.T) {
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
				t.Fatal("browser credentials crossed login boundary")
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "email=reader%40example.test") {
				t.Fatalf("login body = %q", body)
			}
			http.SetCookie(w, &http.Cookie{Name: "hydro_session", Value: "core-secret", Path: "/", HttpOnly: true})
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path == "/auth/logout" {
			if r.Header.Get("Cookie") != "hydro_session=core-secret" {
				t.Fatalf("logout cookie = %q", r.Header.Get("Cookie"))
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer core.Close()
	origin, _ := url.Parse(core.URL)
	store, err := bffsession.New(bffsession.Config{CookieName: "bff_sid", TTL: time.Minute, MaxEntries: 2, Secure: true})
	if err != nil {
		t.Fatal(err)
	}
	handler := AuthHandler{CoreOrigin: origin, Store: store}
	login := httptest.NewRequest(http.MethodPost, "https://frontend.example/login", strings.NewReader("email=reader%40example.test&password=secret"))
	login.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	login.Header.Set("Cookie", "attacker=should-not-forward")
	login.Header.Set("Authorization", "Bearer attacker")
	loginResponse := httptest.NewRecorder()
	handler.Login(loginResponse, login)
	if loginResponse.Code != http.StatusBadRequest {
		t.Fatalf("credential-bearing login status = %d", loginResponse.Code)
	}
	login.Header.Del("Cookie")
	login.Header.Del("Authorization")
	loginResponse = httptest.NewRecorder()
	handler.Login(loginResponse, login)
	if loginResponse.Code != http.StatusOK || !strings.Contains(loginResponse.Header().Get("Set-Cookie"), "bff_sid=") {
		t.Fatalf("login status=%d cookie=%q body=%s", loginResponse.Code, loginResponse.Header().Get("Set-Cookie"), loginResponse.Body.String())
	}
	var result struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(loginResponse.Body.Bytes(), &result); err != nil || result.CSRFToken == "" {
		t.Fatalf("login response = %s", loginResponse.Body.String())
	}
	setCookie := loginResponse.Header().Get("Set-Cookie")
	cookiePair := setCookie[:strings.Index(setCookie, ";")]
	logout := httptest.NewRequest(http.MethodPost, "https://frontend.example/logout", nil)
	logout.Header.Set("Cookie", cookiePair)
	logout.Header.Set("X-CSRF-Token", result.CSRFToken)
	logoutResponse := httptest.NewRecorder()
	handler.Logout(logoutResponse, logout)
	if logoutResponse.Code != http.StatusNoContent || !strings.Contains(logoutResponse.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("logout status=%d cookie=%q", logoutResponse.Code, logoutResponse.Header().Get("Set-Cookie"))
	}
}

func TestAuthHandlerRejectsMissingCoreCookie(t *testing.T) {
	core := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer core.Close()
	origin, _ := url.Parse(core.URL)
	store, _ := bffsession.New(bffsession.Config{CookieName: "bff_sid", TTL: time.Minute, MaxEntries: 1})
	request := httptest.NewRequest(http.MethodPost, "https://frontend.example/login", strings.NewReader("email=a&password=b"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	(AuthHandler{CoreOrigin: origin, Store: store}).Login(response, request)
	if response.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadGateway)
	}
}
