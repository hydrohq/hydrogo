package security

import "net/http"

// TokenHandler returns the current request's CSRF token as JSON. It is meant
// to run behind the normal CSRFMiddleware (not behind a skipCookieFor route),
// so a request here always receives a fresh Set-Cookie if it didn't already
// carry one. Public, cache-eligible pages that render a form needing a token
// but are themselves excluded from minting a fresh cookie (see CSRFMiddleware's
// skipCookieFor) fetch it from here client-side instead of embedding it in
// server-rendered, shared/cached HTML.
func TokenHandler(w http.ResponseWriter, r *http.Request) {
	token := CSRFTokenFromContext(r.Context())
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(`{"token":"` + token + `"}`))
}
