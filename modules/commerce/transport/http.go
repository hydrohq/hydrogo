package transport

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

type Handler struct {
	DB       database.Executor
	Commerce commerce.Service
	Catalog  commerce.CatalogService
}

type PaymentWebhookHandler struct{ Service commerce.PaymentService }
type checkoutInput struct {
	Items []commerce.CheckoutItem `json:"items"`
}

type Route struct {
	Pattern   string
	Handler   http.Handler
	Protected bool
}

func (h Handler) Routes() []Route {
	return []Route{
		{Pattern: "GET /api/commerce/products", Handler: http.HandlerFunc(h.products), Protected: false},
		{Pattern: "POST /api/commerce/products", Handler: http.HandlerFunc(h.createProduct), Protected: true},
		{Pattern: "POST /api/commerce/cart/checkout", Handler: http.HandlerFunc(h.checkout), Protected: true},
	}
}

type productInput struct {
	SKU        string                 `json:"sku"`
	Title      string                 `json:"title"`
	PriceCents int64                  `json:"price_cents"`
	Currency   string                 `json:"currency"`
	Stock      int                    `json:"stock"`
	Status     commerce.ProductStatus `json:"status"`
}

func (h Handler) products(w http.ResponseWriter, r *http.Request) {
	products, err := h.Catalog.ListProducts(r.Context(), h.DB, true)
	if err != nil {
		http.Error(w, "catalog unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(products)
}

func (h Handler) createProduct(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var input productInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		http.Error(w, "invalid product", http.StatusBadRequest)
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "catalog unavailable", http.StatusServiceUnavailable)
		return
	}
	product := commerce.Product{ID: id, SKU: input.SKU, Title: input.Title, PriceCents: input.PriceCents, Currency: input.Currency, Stock: input.Stock, Status: input.Status}
	if err := h.Catalog.CreateProduct(r.Context(), product); err != nil {
		http.Error(w, "invalid product", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(product)
}

func (h PaymentWebhookHandler) Routes() []Route {
	return []Route{{Pattern: "POST /api/webhooks/payment/{provider}", Handler: http.HandlerFunc(h.webhook)}}
}

func (h PaymentWebhookHandler) webhook(w http.ResponseWriter, r *http.Request) {
	if err := h.Service.HandleWebhook(r.Context(), r.PathValue("provider"), r); err != nil {
		http.Error(w, "invalid payment webhook", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) checkout(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		http.Error(w, "Idempotency-Key is required", http.StatusBadRequest)
		return
	}
	var input checkoutInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		http.Error(w, "invalid checkout", http.StatusBadRequest)
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "checkout unavailable", http.StatusServiceUnavailable)
		return
	}
	order, err := h.Commerce.PlaceOrder(r.Context(), commerce.Checkout{ID: id, UserID: user.ID, IdempotencyKey: key, Items: input.Items})
	if err != nil {
		http.Error(w, "checkout failed", http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(order)
}

func newID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}
