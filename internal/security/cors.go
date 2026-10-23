package security

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// NewPublicReadCORSMiddleware permits selected, published GET/HEAD API
// projections to an explicit origin allowlist. It never sets credentialed CORS
// headers and deliberately does not handle preflight requests: browser writes
// remain same-site or BFF-only until a separate session contract exists.
func NewPublicReadCORSMiddleware(origins []string) (func(http.Handler) http.Handler, error) {
	allowed := make(map[string]struct{}, len(origins))
	for _, value := range origins {
		origin, err := normalizeOrigin(value)
		if err != nil {
			return nil, err
		}
		if _, exists := allowed[origin]; exists {
			return nil, fmt.Errorf("duplicate public API CORS origin %q", value)
		}
		allowed[origin] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(allowed) > 0 && (r.Method == http.MethodGet || r.Method == http.MethodHead) && isPublicReadAPIPath(r.URL.Path) {
				if origin, err := normalizeOrigin(r.Header.Get("Origin")); err == nil {
					if _, ok := allowed[origin]; ok {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Add("Vary", "Origin")
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}

// Catalog reads (products, variants, categories) go through the generic
// content-type API, not a commerce-specific path -- see
// content-types/commerce/*.yaml. commerce.stock and commerce.orders/{id}
// are the two commerce GETs with no cookie identity behind them (stock is
// not shopper-specific; an order read is scoped by its own unguessable id,
// not a cookie), so unlike cart they are safe to publish cross-origin for a
// storefront hosted on its own domain. GET /api/v1/commerce/cart is
// cookie-identity-scoped and stays out: SameSite=Lax would not carry that
// cookie cross-origin anyway.
func isPublicReadAPIPath(path string) bool {
	if path == "/api/v1/pages" || path == "/api/v1/pages/by-slug" || path == "/api/v1/news" || path == "/api/v1/news/by-slug" {
		return true
	}
	return strings.HasPrefix(path, "/api/v1/commerce/stock/") || strings.HasPrefix(path, "/api/v1/commerce/orders/")
}

func normalizeOrigin(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("origin is empty")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid origin %q", value)
	}
	host, err := canonicalHostname(parsed.Hostname())
	if err != nil {
		return "", err
	}
	port := parsed.Port()
	if port != "" && !(parsed.Scheme == "https" && port == "443") && !(parsed.Scheme == "http" && port == "80") {
		host += ":" + port
	}
	return parsed.Scheme + "://" + host, nil
}
