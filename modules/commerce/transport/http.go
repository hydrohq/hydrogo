// Package transport exposes modules/commerce over HTTP: a public cart and
// checkout API (an anonymous visitor needs no account, see cookie.go) and a
// small protected stock-management endpoint for admins. Product/variant
// catalog reads go through the generic content-type public API
// (/api/v1/content/products) already, not through here.
package transport

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

// Route mirrors modules/news/transport.Route: Protected false is translated
// to router.Public centrally in cmd/hydro/routes.go, the same way every
// other module's routes are.
type Route struct {
	Pattern   string
	Handler   http.Handler
	Protected bool
}

const cartCookieName = "hydro_commerce_cart"

// Handler wires modules/commerce.Service to HTTP. MemberID is optional: a
// project that also enables modules/members can supply a resolver so a
// logged-in visitor's cart is tied to their account instead of an anonymous
// cookie; commerce does not import modules/members directly (no module
// dependency edge needed for that), so the resolver is injected by
// cmd/hydro at wiring time.
type Handler struct {
	Service            commerce.Service
	DB                 database.Executor
	Secure             bool
	DefaultCurrency    string
	CartLimiter        security.RateLimiter
	CheckoutLimiter    security.RateLimiter
	RequireTenantScope bool
	MemberID           func(*http.Request) (string, bool)
}

func (h Handler) defaultCurrency() string {
	if h.DefaultCurrency != "" {
		return h.DefaultCurrency
	}
	return "TRY"
}

func (h Handler) memberID(r *http.Request) string {
	if h.MemberID == nil {
		return ""
	}
	id, ok := h.MemberID(r)
	if !ok {
		return ""
	}
	return id
}

// identity resolves which cart the request belongs to: a logged-in
// member's, or an anonymous visitor's cookie -- minting and setting a new
// one if this is their first cart action. Exactly one of the two return
// values is non-empty, matching commerce.Cart's own invariant.
func (h Handler) identity(w http.ResponseWriter, r *http.Request) (memberID, anonymousToken string, err error) {
	if id := h.memberID(r); id != "" {
		return id, "", nil
	}
	if cookie, cookieErr := r.Cookie(cartCookieName); cookieErr == nil && cookie.Value != "" {
		return "", cookie.Value, nil
	}
	token, err := randomToken()
	if err != nil {
		return "", "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name: cartCookieName, Value: token, Path: "/", MaxAge: int((30 * 24 * time.Hour).Seconds()),
		HttpOnly: true, Secure: h.Secure, SameSite: http.SameSiteLaxMode,
	}) // #nosec G124 -- Secure follows the secure_cookies setting (off only for plain-HTTP development); the cookie is HttpOnly and SameSite
	return "", token, nil
}

func randomToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func rateLimited(w http.ResponseWriter, limiter security.RateLimiter, key string) bool {
	if limiter == nil {
		return false
	}
	if limiter.Allow(key) {
		return false
	}
	security.WriteRateLimitedResponse(w, "too many requests", time.Minute)
	return true
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return r.RemoteAddr
}

// Routes returns the public cart/checkout API and the protected stock
// endpoint.
func (h Handler) Routes() []Route {
	return []Route{
		{Pattern: "GET /api/v1/commerce/cart", Handler: http.HandlerFunc(h.getCart)},
		{Pattern: "POST /api/v1/commerce/cart/items", Handler: http.HandlerFunc(h.addItem)},
		{Pattern: "DELETE /api/v1/commerce/cart/items/{id}", Handler: http.HandlerFunc(h.removeItem)},
		{Pattern: "POST /api/v1/commerce/checkout", Handler: http.HandlerFunc(h.checkout)},
		{Pattern: "POST /api/v1/admin/commerce/stock", Handler: http.HandlerFunc(h.setStock), Protected: true},
	}
}

type cartResponse struct {
	ID    string         `json:"id"`
	Items []cartItemView `json:"items"`
}

type cartItemView struct {
	ID             string `json:"id"`
	VariantID      string `json:"variant_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
}

func (h Handler) getCart(w http.ResponseWriter, r *http.Request) {
	memberID, anonymousToken, err := h.identity(w, r)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "could not resolve cart identity", nil)
		return
	}
	cart, err := h.Service.GetOrCreateCart(r.Context(), h.DB, memberID, anonymousToken, h.defaultCurrency())
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	writeCart(w, r, h, cart)
}

type addItemRequest struct {
	VariantID      string `json:"variant_id"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
}

func (h Handler) addItem(w http.ResponseWriter, r *http.Request) {
	memberID, anonymousToken, err := h.identity(w, r)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "could not resolve cart identity", nil)
		return
	}
	if rateLimited(w, h.CartLimiter, memberID+anonymousToken+clientIP(r)) {
		return
	}
	var input addItemRequest
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	if strings.TrimSpace(input.VariantID) == "" || input.Quantity <= 0 || input.UnitPriceCents < 0 {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "variant_id, a positive quantity and a non-negative unit_price_cents are required", nil)
		return
	}
	cart, err := h.Service.GetOrCreateCart(r.Context(), h.DB, memberID, anonymousToken, h.defaultCurrency())
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	if err := h.Service.AddItem(r.Context(), h.DB, cart.ID, input.VariantID, input.Quantity, input.UnitPriceCents); err != nil {
		api.WriteError(w, r, http.StatusUnprocessableEntity, err.Error(), nil)
		return
	}
	writeCart(w, r, h, cart)
}

func (h Handler) removeItem(w http.ResponseWriter, r *http.Request) {
	memberID, anonymousToken, err := h.identity(w, r)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "could not resolve cart identity", nil)
		return
	}
	cart, err := h.Service.GetOrCreateCart(r.Context(), h.DB, memberID, anonymousToken, h.defaultCurrency())
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	if err := h.Service.Repository.RemoveItem(r.Context(), h.DB, cart.ID, r.PathValue("id")); err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	writeCart(w, r, h, cart)
}

func writeCart(w http.ResponseWriter, r *http.Request, h Handler, cart commerce.Cart) {
	items, err := h.Service.Repository.ListItems(r.Context(), h.DB, cart.ID)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	views := make([]cartItemView, 0, len(items))
	for _, item := range items {
		views = append(views, cartItemView{ID: item.ID, VariantID: item.VariantID, Quantity: item.Quantity, UnitPriceCents: item.UnitPriceCents})
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, cartResponse{ID: cart.ID, Items: views}, nil)
}

type checkoutRequest struct {
	GuestName       string `json:"guest_name"`
	GuestEmail      string `json:"guest_email"`
	GuestPhone      string `json:"guest_phone"`
	GuestAddress    string `json:"guest_address"`
	GuestCity       string `json:"guest_city"`
	GuestCountry    string `json:"guest_country"`
	GuestPostalCode string `json:"guest_postal_code"`
	IdempotencyKey  string `json:"idempotency_key"`
}

func (h Handler) checkout(w http.ResponseWriter, r *http.Request) {
	memberID, anonymousToken, err := h.identity(w, r)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "could not resolve cart identity", nil)
		return
	}
	if rateLimited(w, h.CheckoutLimiter, memberID+anonymousToken+clientIP(r)) {
		return
	}
	var input checkoutRequest
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "idempotency_key is required", nil)
		return
	}
	if memberID == "" && strings.TrimSpace(input.GuestEmail) == "" {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "guest_email is required for a guest checkout", nil)
		return
	}
	cart, err := h.Service.GetOrCreateCart(r.Context(), h.DB, memberID, anonymousToken, h.defaultCurrency())
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	order, err := h.Service.Checkout(r.Context(), commerce.CheckoutRequest{
		CartID: cart.ID, MemberID: memberID,
		GuestName: input.GuestName, GuestEmail: input.GuestEmail, GuestPhone: input.GuestPhone,
		GuestAddress: input.GuestAddress, GuestCity: input.GuestCity, GuestCountry: input.GuestCountry, GuestPostalCode: input.GuestPostalCode,
		IdempotencyKey: input.IdempotencyKey,
	})
	if err != nil {
		if errors.Is(err, commerce.ErrOutOfStock) {
			api.WriteError(w, r, http.StatusConflict, "one or more items in your cart are no longer available in the requested quantity", nil)
			return
		}
		api.WriteError(w, r, http.StatusUnprocessableEntity, err.Error(), nil)
		return
	}
	api.WriteData(w, http.StatusCreated, order, nil)
}

type setStockRequest struct {
	VariantID string `json:"variant_id"`
	Available int    `json:"available"`
}

func (h Handler) setStock(w http.ResponseWriter, r *http.Request) {
	var input setStockRequest
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	if strings.TrimSpace(input.VariantID) == "" || input.Available < 0 {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "variant_id and a non-negative available are required", nil)
		return
	}
	if err := h.Service.Repository.EnsureStock(r.Context(), h.DB, input.VariantID, input.Available); err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	stock, err := h.Service.Repository.GetStock(r.Context(), h.DB, input.VariantID)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, err.Error(), nil)
		return
	}
	api.WriteData(w, http.StatusOK, stock, nil)
}
