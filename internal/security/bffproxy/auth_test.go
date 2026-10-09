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
	"github.com/ardaozyildirim/hydrogo/internal/testhttp"
)

type loginLimiterStub struct {
	allow bool
	keys  []string
}

func (l *loginLimiterStub) Allow(key string) bool {
	l.keys = append(l.keys, key)
	return l.allow
}

func TestAuthHandlerLoginAndLogoutExchange(t *testing.T) {
	const coreURL = "http://127.0.0.1:19201"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19201": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	})})
	origin, _ := url.Parse(coreURL)
	store, err := bffsession.New(bffsession.Config{CookieName: "bff_sid", TTL: time.Minute, MaxEntries: 2, Secure: true})
	if err != nil {
		t.Fatal(err)
	}
	handler := AuthHandler{CoreOrigin: origin, Client: client, Store: store}
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
	const coreURL = "http://127.0.0.1:19202"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19202": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })})
	origin, _ := url.Parse(coreURL)
	store, _ := bffsession.New(bffsession.Config{CookieName: "bff_sid", TTL: time.Minute, MaxEntries: 1})
	request := httptest.NewRequest(http.MethodPost, "https://frontend.example/login", strings.NewReader("email=a&password=b"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	(AuthHandler{CoreOrigin: origin, Client: client, Store: store}).Login(response, request)
	if response.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadGateway)
	}
}

func TestAuthHandlerLoginLimiterBlocksBeforeCore(t *testing.T) {
	coreCalls := 0
	const coreURL = "http://127.0.0.1:19203"
	client := testhttp.Client(testhttp.Handlers{"127.0.0.1:19203": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coreCalls++
		http.SetCookie(w, &http.Cookie{Name: "hydro_session", Value: "should-not-be-reached"})
		w.WriteHeader(http.StatusOK)
	})})
	origin, _ := url.Parse(coreURL)
	store, err := bffsession.New(bffsession.Config{CookieName: "bff_sid", TTL: time.Minute, MaxEntries: 1})
	if err != nil {
		t.Fatal(err)
	}
	limiter := &loginLimiterStub{}
	handler := AuthHandler{CoreOrigin: origin, Client: client, Store: store, LoginLimiter: limiter}
	request := httptest.NewRequest(http.MethodPost, "https://frontend.example/login", strings.NewReader("email=Reader%40Example.test&password=secret"))
	request.RemoteAddr = "203.0.113.9:43122"
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	handler.Login(response, request)
	if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") != "60" {
		t.Fatalf("status=%d retry-after=%q", response.Code, response.Header().Get("Retry-After"))
	}
	if coreCalls != 0 {
		t.Fatalf("core calls = %d, want 0", coreCalls)
	}
	if len(limiter.keys) != 1 || limiter.keys[0] != "reader@example.test\x00203.0.113.9" {
		t.Fatalf("limiter key = %#v", limiter.keys)
	}
}
