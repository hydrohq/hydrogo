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
}
type checkoutInput struct {
	Items []commerce.CheckoutItem `json:"items"`
}

type Route struct {
	Pattern   string
	Handler   http.Handler
	Protected bool
}

func (h Handler) Routes() []Route {
	return []Route{{Pattern: "POST /api/commerce/cart/checkout", Handler: http.HandlerFunc(h.checkout), Protected: true}}
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
