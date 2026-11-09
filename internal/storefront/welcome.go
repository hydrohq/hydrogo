package storefront

import (
	_ "embed"
	"net/http"
)

//go:embed welcome.html
var welcomeHTML []byte

// RenderWelcome writes the neutral page a project shows at "/" when it has no
// tenant storefront: it says the server is up and points to the admin and API.
func RenderWelcome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'")
	_, _ = w.Write(welcomeHTML)
}
