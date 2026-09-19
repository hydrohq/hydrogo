package bffsession

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	store, err := New(Config{CookieName: "bff_sid", TTL: time.Minute, MaxEntries: 2, Secure: true})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func requestWithCookie(t *testing.T, header string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "https://frontend.example/api/v1/member", nil)
	r.Header.Set("Cookie", header)
	return r
}

func TestStoreKeepsCoreCookieServerSideAndValidatesCSRF(t *testing.T) {
	store := testStore(t)
	response := httptest.NewRecorder()
	csrf, err := store.Create(response, "hydro_session=core-secret")
	if err != nil {
		t.Fatal(err)
	}
	setCookie := response.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, "bff_sid=") || !strings.Contains(setCookie, "HttpOnly") || !strings.Contains(setCookie, "Secure") {
		t.Fatalf("unexpected cookie: %q", setCookie)
	}
	request := requestWithCookie(t, setCookie[strings.Index(setCookie, "bff_sid="):strings.Index(setCookie, ";")])
	coreCookie, err := store.CoreCookie(request.Context(), request)
	if err != nil || coreCookie != "hydro_session=core-secret" {
		t.Fatalf("core cookie = %q, err = %v", coreCookie, err)
	}
	if err := store.Validate(request.Context(), request); err == nil {
		t.Fatal("missing csrf accepted")
	}
	request.Header.Set("X-CSRF-Token", csrf)
	if err := store.Validate(request.Context(), request); err != nil {
		t.Fatal(err)
	}
}

func TestStoreRevokeAndExpiryFailClosed(t *testing.T) {
	store := testStore(t)
	store.now = func() time.Time { return time.Unix(100, 0) }
	response := httptest.NewRecorder()
	_, err := store.Create(response, "core=session")
	if err != nil {
		t.Fatal(err)
	}
	cookie := response.Header().Get("Set-Cookie")
	request := requestWithCookie(t, cookie[strings.Index(cookie, "bff_sid="):strings.Index(cookie, ";")])
	store.Revoke(nil, request)
	if _, err := store.CoreCookie(request.Context(), request); err == nil {
		t.Fatal("revoked session accepted")
	}
	response = httptest.NewRecorder()
	_, err = store.Create(response, "core=session")
	if err != nil {
		t.Fatal(err)
	}
	store.now = func() time.Time { return time.Unix(161, 0) }
	request = requestWithCookie(t, response.Header().Get("Set-Cookie")[strings.Index(response.Header().Get("Set-Cookie"), "bff_sid="):strings.Index(response.Header().Get("Set-Cookie"), ";")])
	if _, err := store.CoreCookie(request.Context(), request); err == nil {
		t.Fatal("expired session accepted")
	}
}

func TestStoreBoundsCapacity(t *testing.T) {
	store := testStore(t)
	for i := 0; i < 2; i++ {
		if _, err := store.Create(nil, "core=session"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.Create(nil, "core=session"); err != ErrCapacity {
		t.Fatalf("capacity error = %v", err)
	}
}
