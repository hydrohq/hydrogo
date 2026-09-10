package transport

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/internal/ui"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

type Handler struct {
	DB                database.Executor
	Commerce          commerce.Service
	Catalog           commerce.CatalogService
	Cart              commerce.CartService
	CatalogVariants   commerce.VariantStore
	Orders            commerce.OrderReader
	Payments          map[string]commerce.PaymentProvider
	PaymentService    commerce.PaymentService
	AdminCollection   adminstudio.Collection
	VariantCollection adminstudio.Collection
}

type PaymentWebhookHandler struct{ Service commerce.PaymentService }
type checkoutInput struct {
	CartID   string                  `json:"cart_id"`
	Items    []commerce.CheckoutItem `json:"items"`
	Customer paymentCustomerInput    `json:"customer"`
}

type paymentCustomerInput struct {
	Name           string `json:"name"`
	IdentityNumber string `json:"identity_number"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	Address        string `json:"address"`
	City           string `json:"city"`
	Country        string `json:"country"`
	PostalCode     string `json:"postal_code"`
}

func (input paymentCustomerInput) model(clientIP string) commerce.PaymentCustomer {
	return commerce.PaymentCustomer{Name: input.Name, IdentityNumber: input.IdentityNumber, Email: input.Email, Phone: input.Phone, Address: input.Address, City: input.City, Country: input.Country, PostalCode: input.PostalCode, ClientIP: clientIP}
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
		{Pattern: "GET /api/commerce/products/{productID}/variants", Handler: http.HandlerFunc(h.variants), Protected: false},
		{Pattern: "POST /api/commerce/products/{productID}/variants", Handler: http.HandlerFunc(h.createVariant), Protected: true},
		{Pattern: "GET /api/commerce/cart", Handler: http.HandlerFunc(h.cart), Protected: true},
		{Pattern: "POST /api/commerce/cart/items", Handler: http.HandlerFunc(h.addCartItem), Protected: true},
		{Pattern: "DELETE /api/commerce/cart/items/{productID}", Handler: http.HandlerFunc(h.removeCartItem), Protected: true},
		{Pattern: "POST /api/commerce/cart/checkout", Handler: http.HandlerFunc(h.checkout), Protected: true},
		{Pattern: "POST /api/commerce/orders/{orderID}/payment-intent", Handler: http.HandlerFunc(h.paymentIntent), Protected: true},
		{Pattern: "GET /shop", Handler: http.HandlerFunc(h.shop), Protected: false},
		{Pattern: "GET /shop/cart", Handler: http.HandlerFunc(h.cartPage), Protected: true},
		{Pattern: "POST /shop/cart/items", Handler: http.HandlerFunc(h.addCartItemForm), Protected: true},
		{Pattern: "POST /shop/cart/items/{productID}/remove", Handler: http.HandlerFunc(h.removeCartItemForm), Protected: true},
		{Pattern: "GET /shop/checkout", Handler: http.HandlerFunc(h.checkoutPage), Protected: true},
		{Pattern: "POST /shop/checkout", Handler: http.HandlerFunc(h.checkoutForm), Protected: true},
		{Pattern: "GET /admin/commerce/products", Handler: http.HandlerFunc(h.adminProducts), Protected: true},
		{Pattern: "POST /admin/commerce/products", Handler: http.HandlerFunc(h.adminCreateProduct), Protected: true},
		{Pattern: "GET /admin/commerce/products/{productID}/variants", Handler: http.HandlerFunc(h.adminVariants), Protected: true},
		{Pattern: "POST /admin/commerce/products/{productID}/variants", Handler: http.HandlerFunc(h.adminCreateVariant), Protected: true},
		{Pattern: "GET /admin/commerce/orders", Handler: http.HandlerFunc(h.adminOrders), Protected: true},
		{Pattern: "POST /admin/commerce/orders/{orderID}/refund", Handler: http.HandlerFunc(h.adminRefund), Protected: true},
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

type cartItemInput struct {
	ProductID string `json:"product_id"`
	VariantID string `json:"variant_id"`
	Quantity  int    `json:"quantity"`
}

type variantInput struct {
	SKU        string                 `json:"sku"`
	Title      string                 `json:"title"`
	PriceCents int64                  `json:"price_cents"`
	Stock      int                    `json:"stock"`
	Status     commerce.ProductStatus `json:"status"`
}

type paymentIntentInput struct {
	Provider string `json:"provider"`
}

func (h Handler) paymentIntent(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || h.Orders == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var input paymentIntentInput
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid payment request", http.StatusBadRequest)
			return
		}
		input.Provider = r.FormValue("provider")
	} else if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil && err != io.EOF && err != http.ErrBodyReadAfterClose {
		http.Error(w, "invalid payment request", http.StatusBadRequest)
		return
	}
	providerName := input.Provider
	if providerName == "" {
		providerName = "stripe"
	}
	provider, ok := h.Payments[providerName]
	if !ok || provider == nil {
		http.Error(w, "payment provider is not configured", http.StatusServiceUnavailable)
		return
	}
	order, err := h.Orders.GetOrder(r.Context(), h.DB, r.PathValue("orderID"))
	if err != nil {
		http.Error(w, "order unavailable", http.StatusNotFound)
		return
	}
	if order.UserID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if order.PaymentStatus != "pending" {
		http.Error(w, "order is not payable", http.StatusConflict)
		return
	}
	intent, err := provider.CreatePaymentIntent(r.Context(), order)
	if err != nil {
		http.Error(w, "payment intent unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(intent)
}

func (h Handler) variants(w http.ResponseWriter, r *http.Request) {
	if h.CatalogVariants == nil {
		http.Error(w, "variant catalog unavailable", http.StatusServiceUnavailable)
		return
	}
	variants, err := h.Catalog.ListVariants(r.Context(), h.DB, r.PathValue("productID"), true)
	if err != nil {
		http.Error(w, "variant catalog unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(variants)
}

func (h Handler) createVariant(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.CatalogVariants == nil {
		http.Error(w, "variant catalog unavailable", http.StatusServiceUnavailable)
		return
	}
	var input variantInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		http.Error(w, "invalid variant", http.StatusBadRequest)
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "catalog unavailable", http.StatusServiceUnavailable)
		return
	}
	variant := commerce.ProductVariant{ID: id, ProductID: r.PathValue("productID"), SKU: input.SKU, Title: input.Title, PriceCents: input.PriceCents, Stock: input.Stock, Status: input.Status}
	if err := h.Catalog.CreateVariant(r.Context(), variant); err != nil {
		http.Error(w, "invalid variant", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(variant)
}

func (h Handler) cart(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	cart, err := h.Cart.Get(r.Context(), h.DB, user.ID)
	if err != nil {
		http.Error(w, "cart unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(cart)
}

func (h Handler) addCartItem(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var input cartItemInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		http.Error(w, "invalid cart item", http.StatusBadRequest)
		return
	}
	cart, err := h.Cart.Add(r.Context(), user.ID, input.ProductID, input.VariantID, input.Quantity)
	if err != nil {
		http.Error(w, "cannot add cart item", http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(cart)
}

func (h Handler) removeCartItem(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := h.Cart.Remove(r.Context(), user.ID, r.PathValue("productID")); err != nil {
		http.Error(w, "cannot remove cart item", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func (h Handler) shop(w http.ResponseWriter, r *http.Request) {
	products, err := h.Catalog.ListProducts(r.Context(), h.DB, true)
	if err != nil {
		http.Error(w, "shop unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	user := auth.UserFromContext(r.Context())
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Shop</title><link rel="stylesheet" href="/theme/static/theme.css?v=20260910"></head><body><main class="theme-corporate-content shop-page"><nav class="shop-nav"><a href="/shop">HydroGo Shop</a><a href="/shop/cart">Cart</a></nav><h1>Shop</h1>`))
	if user == nil || !user.Active {
		_, _ = w.Write([]byte(`<aside class="login-notice"><strong>Sepete eklemek için giriş yapın.</strong> <a href="/login?return_to=%2Fshop">Demo hesabıyla giriş</a></aside>`))
	}
	_, _ = w.Write([]byte(`<section class="shop-grid">`))
	for _, product := range products {
		variants := []commerce.ProductVariant(nil)
		if h.CatalogVariants != nil {
			variants, _ = h.Catalog.ListVariants(r.Context(), h.DB, product.ID, true)
		}
		_, _ = fmt.Fprintf(w, `<article class="product-card"><h2>%s</h2><div class="product-description">%s</div><p class="product-price">%s %d</p><form method="post" action="/shop/cart/items"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="product_id" value="%s">`, template.HTMLEscapeString(product.Title), ui.RichTextHTML(product.Description), template.HTMLEscapeString(product.Currency), product.PriceCents, template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), template.HTMLEscapeString(product.ID))
		if len(variants) > 0 {
			_, _ = w.Write([]byte(`<label>Variant <select name="variant_id" required><option value="">Choose a variant</option>`))
			for _, variant := range variants {
				if variant.Status == commerce.ProductOutOfStock || variant.Stock == 0 {
					continue
				}
				_, _ = fmt.Fprintf(w, `<option value="%s">%s — %d %s</option>`, template.HTMLEscapeString(variant.ID), template.HTMLEscapeString(variant.Title), variant.PriceCents, template.HTMLEscapeString(variant.Currency))
			}
			_, _ = w.Write([]byte(`</select></label>`))
		}
		button := `<button type="submit">Add to cart</button>`
		if user == nil || !user.Active {
			button = `<button type="button" disabled title="Login required">Login required</button>`
		}
		_, _ = fmt.Fprintf(w, `<label>Quantity <input name="quantity" type="number" min="1" max="1000" value="1"></label>%s</form></article>`, button)
	}
	_, _ = w.Write([]byte(`</section></main></body></html>`))
}

func (h Handler) addCartItemForm(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid cart item", http.StatusBadRequest)
		return
	}
	quantity, err := strconv.Atoi(r.FormValue("quantity"))
	if err != nil {
		http.Error(w, "invalid quantity", http.StatusBadRequest)
		return
	}
	if _, err := h.Cart.Add(r.Context(), user.ID, r.FormValue("product_id"), r.FormValue("variant_id"), quantity); err != nil {
		http.Error(w, "cannot add cart item", http.StatusConflict)
		return
	}
	http.Redirect(w, r, "/shop/cart", http.StatusSeeOther)
}

func (h Handler) cartPage(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	cart, err := h.Cart.Get(r.Context(), h.DB, user.ID)
	if err != nil {
		http.Error(w, "cart unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Cart</title><link rel="stylesheet" href="/theme/static/theme.css?v=20260910"></head><body><main class="theme-corporate-content shop-page"><a href="/shop">Continue shopping</a><h1>Cart</h1><ul class="cart-list">`))
	for _, item := range cart.Items {
		_, _ = fmt.Fprintf(w, `<li>%s × %d — %d %s <form method="post" action="/shop/cart/items/%s/remove"><input type="hidden" name="_csrf" value="%s"><button type="submit">Remove</button></form></li>`, template.HTMLEscapeString(item.Title), item.Quantity, item.TotalCents, template.HTMLEscapeString(cart.Currency), template.HTMLEscapeString(item.ProductID), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())))
	}
	_, _ = fmt.Fprintf(w, `</ul><p class="cart-total">Total: %d %s</p><a class="theme-corporate-hero" href="/shop/checkout">Checkout</a></main></body></html>`, cart.TotalCents, template.HTMLEscapeString(cart.Currency))
}

func (h Handler) checkoutPage(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	cart, err := h.Cart.Get(r.Context(), h.DB, user.ID)
	if err != nil {
		http.Error(w, "cart unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Checkout</title><link rel="stylesheet" href="/theme/static/theme.css?v=20260910"></head><body><main class="theme-corporate-content shop-page"><h1>Checkout</h1><p>Total: %d %s</p><form method="post" action="/shop/checkout"><input type="hidden" name="_csrf" value="%s"><label>Name <input name="customer_name" required maxlength="120"></label><label>Identity number <input name="customer_identity_number" required maxlength="20"></label><label>Email <input name="customer_email" type="email" required maxlength="254"></label><label>Phone <input name="customer_phone" required maxlength="32"></label><label>Address <textarea name="customer_address" required maxlength="500"></textarea></label><label>City <input name="customer_city" required maxlength="100"></label><label>Country <input name="customer_country" value="TR" required maxlength="2"></label><label>Postal code <input name="customer_postal_code" maxlength="20"></label><button type="submit">Place order</button></form><a href="/shop/cart">Back to cart</a></main></body></html>`, cart.TotalCents, template.HTMLEscapeString(cart.Currency), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())))
}

func (h Handler) checkoutForm(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid checkout", http.StatusBadRequest)
		return
	}
	cart, err := h.Cart.Get(r.Context(), h.DB, user.ID)
	if err != nil || cart.ID == "" || len(cart.Items) == 0 {
		http.Error(w, "cart is empty", http.StatusConflict)
		return
	}
	orderID, err := newID()
	if err != nil {
		http.Error(w, "checkout unavailable", http.StatusServiceUnavailable)
		return
	}
	key, err := newID()
	if err != nil {
		http.Error(w, "checkout unavailable", http.StatusServiceUnavailable)
		return
	}
	order, err := h.Commerce.PlaceOrder(r.Context(), commerce.Checkout{ID: orderID, UserID: user.ID, CartID: cart.ID, IdempotencyKey: key, Customer: commerce.PaymentCustomer{Name: r.FormValue("customer_name"), IdentityNumber: r.FormValue("customer_identity_number"), Email: r.FormValue("customer_email"), Phone: r.FormValue("customer_phone"), Address: r.FormValue("customer_address"), City: r.FormValue("customer_city"), Country: r.FormValue("customer_country"), PostalCode: r.FormValue("customer_postal_code"), ClientIP: requestClientIP(r)}})
	if err != nil {
		http.Error(w, "checkout failed", http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Order received</title></head><body><h1>Order received</h1><p>Order ID: <code>%s</code></p><p>Status: %s</p><p>Payment is pending.</p><form id="payment-form"><input type="hidden" id="csrf-token" value="%s"><label>Provider <select id="payment-provider"><option value="stripe">Stripe</option><option value="iyzico">iyzico</option><option value="paytr">PayTR</option></select></label><button id="payment-submit" type="submit">Start payment</button></form><p id="payment-status" role="status"></p><pre id="payment-token" hidden></pre><a href="/shop">Back to shop</a><script>(function(){const form=document.getElementById('payment-form');const button=document.getElementById('payment-submit');const status=document.getElementById('payment-status');const token=document.getElementById('payment-token');form.addEventListener('submit',async function(event){event.preventDefault();button.disabled=true;status.textContent='Starting payment…';token.hidden=true;try{const response=await fetch('/api/commerce/orders/%s/payment-intent',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':document.getElementById('csrf-token').value},body:JSON.stringify({provider:document.getElementById('payment-provider').value})});const data=await response.json().catch(function(){return {};});if(!response.ok)throw new Error(data.error||'Payment intent unavailable');if(data.redirect_url){window.location.assign(data.redirect_url);return;}status.textContent='Payment intent created. Continue with the selected provider.';token.textContent=data.client_token||data.id||'';token.hidden=!token.textContent;}catch(error){status.textContent=error.message||'Payment could not be started.';button.disabled=false;}})})();</script></body></html>`, template.HTMLEscapeString(order.ID), template.HTMLEscapeString(order.PaymentStatus), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), template.HTMLEscapeString(order.ID))
}

func (h Handler) adminProducts(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	products, err := h.Catalog.ListProducts(r.Context(), h.DB, false)
	if err != nil {
		http.Error(w, "catalog unavailable", http.StatusServiceUnavailable)
		return
	}
	collection := h.AdminCollection
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Commerce katalog", "Commerce", security.NonceFromContext(r.Context()), "commerce"); err != nil {
		return
	}
	_, _ = w.Write([]byte(`<section class="studio-card"><h2>Ürün kataloğu</h2><p>Ürünleri, fiyatları ve stok durumunu yönetin.</p><form class="studio-form" method="post" action="/admin/commerce/products">`))
	if err := adminstudio.RenderCreateForm(w, adminstudio.FormData{Collection: collection, Action: "/admin/commerce/products", CSRFToken: security.CSRFTokenFromContext(r.Context()), Nonce: security.NonceFromContext(r.Context()), Values: map[string]string{"currency": "TRY", "status": "draft"}}); err != nil {
		http.Error(w, "admin studio catalog unavailable", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte(`</form></section><section class="studio-card"><h2>Ürünler</h2><div class="table-wrap">`))
	rows := make([]adminstudio.Row, 0, len(products))
	for _, product := range products {
		rows = append(rows, adminstudio.Row{Values: map[string]string{"title": product.Title, "sku": product.SKU, "price_cents": strconv.FormatInt(product.PriceCents, 10), "currency": product.Currency, "stock": strconv.Itoa(product.Stock), "status": string(product.Status)}, Actions: []adminstudio.Action{{Label: "Variants", Href: "/admin/commerce/products/" + product.ID + "/variants"}}})
	}
	if err := adminstudio.RenderList(w, adminstudio.ListData{Collection: collection, Rows: rows}); err != nil {
		http.Error(w, "admin studio catalog unavailable", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte(`</div></section>`))
	_ = ui.AdminEnd(w)
}

func (h Handler) adminVariants(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin || h.CatalogVariants == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	variants, err := h.Catalog.ListVariants(r.Context(), h.DB, r.PathValue("productID"), false)
	if err != nil {
		http.Error(w, "variants unavailable", http.StatusServiceUnavailable)
		return
	}
	var form bytes.Buffer
	if err := adminstudio.RenderCreateForm(&form, adminstudio.FormData{Collection: h.VariantCollection, Action: "/admin/commerce/products/" + r.PathValue("productID") + "/variants", CSRFToken: security.CSRFTokenFromContext(r.Context()), Nonce: security.NonceFromContext(r.Context()), Values: map[string]string{"status": "draft"}, SubmitLabel: "Varyant oluştur"}); err != nil {
		http.Error(w, "admin studio variants unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Varyantlar", "Commerce", security.NonceFromContext(r.Context()), "commerce"); err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, `<section class="studio-card"><a href="/admin/commerce/products">← Kataloğa dön</a><h2>Ürün varyantları</h2><p>SKU, fiyat ve stok seçeneklerini yönetin.</p><form class="studio-form" method="post" action="/admin/commerce/products/%s/variants">%s</form></section><section class="studio-card"><h2>Varyant listesi</h2><div class="table-wrap">`, template.HTMLEscapeString(r.PathValue("productID")), form.String())
	rows := make([]adminstudio.Row, 0, len(variants))
	for _, variant := range variants {
		rows = append(rows, adminstudio.Row{Values: map[string]string{"sku": variant.SKU, "title": variant.Title, "price_cents": strconv.FormatInt(variant.PriceCents, 10), "currency": variant.Currency, "stock": strconv.Itoa(variant.Stock), "status": string(variant.Status)}})
	}
	if err := adminstudio.RenderList(w, adminstudio.ListData{Collection: h.VariantCollection, Rows: rows}); err != nil {
		http.Error(w, "admin studio variants unavailable", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte(`</div></section>`))
	_ = ui.AdminEnd(w)
}

func (h Handler) adminCreateVariant(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin || h.CatalogVariants == nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid variant", http.StatusBadRequest)
		return
	}
	price, err1 := strconv.ParseInt(r.FormValue("price_cents"), 10, 64)
	stock, err2 := strconv.Atoi(r.FormValue("stock"))
	if err1 != nil || err2 != nil {
		http.Error(w, "invalid variant", http.StatusBadRequest)
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "catalog unavailable", http.StatusServiceUnavailable)
		return
	}
	variant := commerce.ProductVariant{ID: id, ProductID: r.PathValue("productID"), SKU: r.FormValue("sku"), Title: r.FormValue("title"), PriceCents: price, Stock: stock, Status: commerce.ProductStatus(r.FormValue("status"))}
	if err := h.Catalog.CreateVariant(r.Context(), variant); err != nil {
		http.Error(w, "invalid variant", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/commerce/products/"+template.URLQueryEscaper(variant.ProductID)+"/variants", http.StatusSeeOther)
}

func (h Handler) adminCreateProduct(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid product", http.StatusBadRequest)
		return
	}
	price, err1 := strconv.ParseInt(r.FormValue("price_cents"), 10, 64)
	stock, err2 := strconv.Atoi(r.FormValue("stock"))
	if err1 != nil || err2 != nil {
		http.Error(w, "invalid product", http.StatusBadRequest)
		return
	}
	id, err := newID()
	if err != nil {
		http.Error(w, "catalog unavailable", http.StatusServiceUnavailable)
		return
	}
	product := commerce.Product{ID: id, SKU: r.FormValue("sku"), Title: r.FormValue("title"), Description: r.FormValue("description"), PriceCents: price, Currency: r.FormValue("currency"), Stock: stock, Status: commerce.ProductStatus(r.FormValue("status"))}
	if err := h.Catalog.CreateProduct(r.Context(), product); err != nil {
		http.Error(w, "invalid product", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/commerce/products", http.StatusSeeOther)
}

func (h Handler) adminOrders(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if h.Orders == nil {
		http.Error(w, "order management unavailable", http.StatusServiceUnavailable)
		return
	}
	orders, err := h.Orders.ListOrders(r.Context(), h.DB, 100)
	if err != nil {
		http.Error(w, "orders unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Siparişler", "Commerce", security.NonceFromContext(r.Context()), "commerce"); err != nil {
		return
	}
	_, _ = w.Write([]byte(`<section class="studio-card"><a href="/admin/commerce/products">← Kataloğa dön</a><h2>Siparişler</h2><p>Ödeme ve teslimat durumlarını izleyin.</p><div class="table-wrap"><table><thead><tr><th>Order</th><th>Customer</th><th>Total</th><th>Payment</th><th>Shipping</th><th>Items</th><th>Refund</th></tr></thead><tbody>`))
	for _, order := range orders {
		refund := `<span>Unavailable</span>`
		if order.PaymentStatus == "paid" {
			refund = fmt.Sprintf(`<form method="post" action="/admin/commerce/orders/%s/refund"><input type="hidden" name="_csrf" value="%s"><input name="provider" placeholder="stripe" required><input name="payment_id" placeholder="provider payment id" required><input name="amount_cents" type="number" min="1" value="%d" required><button type="submit">Refund full order</button></form>`, template.HTMLEscapeString(order.ID), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), order.TotalCents)
		}
		_, _ = fmt.Fprintf(w, `<tr><td><code>%s</code></td><td><code>%s</code></td><td>%d %s</td><td>%s</td><td>%s</td><td>%d</td><td>%s</td></tr>`, template.HTMLEscapeString(order.ID), template.HTMLEscapeString(order.UserID), order.TotalCents, template.HTMLEscapeString(order.Currency), template.HTMLEscapeString(order.PaymentStatus), template.HTMLEscapeString(order.ShippingStatus), len(order.Items), refund)
	}
	_, _ = w.Write([]byte(`</tbody></table></div></section>`))
	_ = ui.AdminEnd(w)
}

func (h Handler) adminRefund(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid refund", http.StatusBadRequest)
		return
	}
	amount, err := strconv.ParseInt(r.FormValue("amount_cents"), 10, 64)
	if err != nil || amount <= 0 {
		http.Error(w, "invalid refund amount", http.StatusBadRequest)
		return
	}
	if err := h.PaymentService.Refund(r.Context(), h.DB, r.PathValue("orderID"), r.FormValue("provider"), r.FormValue("payment_id"), amount); err != nil {
		http.Error(w, "refund failed", http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/admin/commerce/orders", http.StatusSeeOther)
}

func (h Handler) removeCartItemForm(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid cart item", http.StatusBadRequest)
		return
	}
	if err := h.Cart.Remove(r.Context(), user.ID, r.PathValue("productID")); err != nil {
		http.Error(w, "cannot remove cart item", http.StatusConflict)
		return
	}
	http.Redirect(w, r, "/shop/cart", http.StatusSeeOther)
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
	customer := input.Customer.model(requestClientIP(r))
	id, err := newID()
	if err != nil {
		http.Error(w, "checkout unavailable", http.StatusServiceUnavailable)
		return
	}
	order, err := h.Commerce.PlaceOrder(r.Context(), commerce.Checkout{ID: id, UserID: user.ID, IdempotencyKey: key, CartID: input.CartID, Items: input.Items, Customer: customer})
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

func requestClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
