package transport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/hydrohq/hydrogo/internal/api"
	"github.com/hydrohq/hydrogo/modules/commerce"
)

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

// addItemRequest has no price field on purpose: the server prices every
// line from the catalog. A unit_price_cents sent by older storefronts is
// ignored.
type addItemRequest struct {
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity"`
}

func (h Handler) getCart(w http.ResponseWriter, r *http.Request) {
	shopper, err := h.shopper(w, r)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "could not resolve cart identity", nil)
		return
	}
	cart, items, err := h.Service.CartFor(r.Context(), shopper)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeCart(w, cart, items)
}

func (h Handler) addItem(w http.ResponseWriter, r *http.Request) {
	shopper, err := h.shopper(w, r)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "could not resolve cart identity", nil)
		return
	}
	// Keyed on the address alone: the cart cookie is minted by the caller,
	// so a cookie-based key would give every request a fresh budget.
	if rateLimited(w, h.CartLimiter, "cart:"+h.clientIP(r)) {
		return
	}
	var input addItemRequest
	if !api.DecodeJSON(w, r, &input, api.DefaultMaxBodyBytes) {
		return
	}
	maxQuantity := h.MaxLineQuantity
	if maxQuantity <= 0 {
		maxQuantity = 100
	}
	if strings.TrimSpace(input.VariantID) == "" || input.Quantity <= 0 || input.Quantity > maxQuantity {
		api.WriteError(w, r, http.StatusUnprocessableEntity, "variant_id and a quantity between 1 and "+strconv.Itoa(maxQuantity)+" are required", nil)
		return
	}
	cart, items, err := h.Service.AddToCart(r.Context(), shopper, input.VariantID, input.Quantity)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeCart(w, cart, items)
}

func (h Handler) removeItem(w http.ResponseWriter, r *http.Request) {
	shopper, err := h.shopper(w, r)
	if err != nil {
		api.WriteError(w, r, http.StatusInternalServerError, "could not resolve cart identity", nil)
		return
	}
	cart, items, err := h.Service.RemoveFromCart(r.Context(), shopper, r.PathValue("id"))
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	writeCart(w, cart, items)
}

func writeCart(w http.ResponseWriter, cart commerce.Cart, items []commerce.CartItem) {
	views := make([]cartItemView, 0, len(items))
	for _, item := range items {
		views = append(views, cartItemView{ID: item.ID, VariantID: item.VariantID, Quantity: item.Quantity, UnitPriceCents: item.UnitPriceCents})
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, cartResponse{ID: cart.ID, Items: views}, nil)
}

// getStock is a public read of how many units can still be bought. A
// variant nobody stocked yet reads as zero: to a shopper "never stocked" and
// "sold out" look the same.
func (h Handler) getStock(w http.ResponseWriter, r *http.Request) {
	stock, err := h.Service.Stock(r.Context(), r.PathValue("variantID"))
	if err != nil {
		if errors.Is(err, commerce.ErrItemNotFound) {
			api.WriteData(w, http.StatusOK, map[string]any{"variant_id": r.PathValue("variantID"), "available": 0}, nil)
			return
		}
		writeServiceError(w, r, err)
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{"variant_id": stock.VariantID, "available": stock.Available}, nil)
}

// getStockBatch answers GET /stock?ids=a,b,c with {"data":{"stock":{id:available}}}
// so a catalog page can learn every variant's availability in one request
// instead of rendering "in stock" first and correcting it per variant later.
// Unknown variants read as zero, exactly like getStock.
func (h Handler) getStockBatch(w http.ResponseWriter, r *http.Request) {
	const maxIDs = 200
	seen := map[string]bool{}
	out := map[string]int64{}
	for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		if len(seen) >= maxIDs {
			writeServiceError(w, r, commerce.ErrInvalidItem)
			return
		}
		seen[id] = true
		stock, err := h.Service.Stock(r.Context(), id)
		if err != nil {
			if errors.Is(err, commerce.ErrItemNotFound) {
				out[id] = 0
				continue
			}
			writeServiceError(w, r, err)
			return
		}
		out[id] = int64(stock.Available)
	}
	w.Header().Set("Cache-Control", "no-store")
	api.WriteData(w, http.StatusOK, map[string]any{"stock": out}, nil)
}
