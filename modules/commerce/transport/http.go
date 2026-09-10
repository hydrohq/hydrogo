package transport

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/security"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

type Handler struct {
	DB              database.Executor
	Commerce        commerce.Service
	Catalog         commerce.CatalogService
	Cart            commerce.CartService
	CatalogVariants commerce.VariantStore
	Orders          commerce.OrderReader
	Payments        map[string]commerce.PaymentProvider
	PaymentService  commerce.PaymentService
}

type PaymentWebhookHandler struct{ Service commerce.PaymentService }
type checkoutInput struct {
	CartID string                  `json:"cart_id"`
	Items  []commerce.CheckoutItem `json:"items"`
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
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Shop</title></head><body><h1>Shop</h1><a href="/shop/cart">Cart</a><main>`))
	for _, product := range products {
		variants := []commerce.ProductVariant(nil)
		if h.CatalogVariants != nil {
			variants, _ = h.Catalog.ListVariants(r.Context(), h.DB, product.ID, true)
		}
		_, _ = fmt.Fprintf(w, `<article><h2>%s</h2><p>%s %d</p><form method="post" action="/shop/cart/items"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="product_id" value="%s">`, template.HTMLEscapeString(product.Title), template.HTMLEscapeString(product.Currency), product.PriceCents, template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), template.HTMLEscapeString(product.ID))
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
		_, _ = w.Write([]byte(`<label>Quantity <input name="quantity" type="number" min="1" max="1000" value="1"></label><button type="submit">Add to cart</button></form></article>`))
	}
	_, _ = w.Write([]byte(`</main></body></html>`))
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
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Cart</title></head><body><a href="/shop">Continue shopping</a><h1>Cart</h1><ul>`))
	for _, item := range cart.Items {
		_, _ = fmt.Fprintf(w, `<li>%s × %d — %d %s <form method="post" action="/shop/cart/items/%s/remove"><input type="hidden" name="_csrf" value="%s"><button type="submit">Remove</button></form></li>`, template.HTMLEscapeString(item.Title), item.Quantity, item.TotalCents, template.HTMLEscapeString(cart.Currency), template.HTMLEscapeString(item.ProductID), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())))
	}
	_, _ = fmt.Fprintf(w, `</ul><p>Total: %d %s</p><a href="/shop/checkout">Checkout</a></body></html>`, cart.TotalCents, template.HTMLEscapeString(cart.Currency))
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
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Checkout</title></head><body><h1>Checkout</h1><p>Total: %d %s</p><form method="post" action="/shop/checkout"><input type="hidden" name="_csrf" value="%s"><button type="submit">Place order</button></form><a href="/shop/cart">Back to cart</a></body></html>`, cart.TotalCents, template.HTMLEscapeString(cart.Currency), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())))
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
	order, err := h.Commerce.PlaceOrder(r.Context(), commerce.Checkout{ID: orderID, UserID: user.ID, CartID: cart.ID, IdempotencyKey: key})
	if err != nil {
		http.Error(w, "checkout failed", http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Order received</title></head><body><h1>Order received</h1><p>Order ID: <code>%s</code></p><p>Status: %s</p><p>Payment is pending.</p><form id="payment-form"><input type="hidden" id="csrf-token" value="%s"><label>Provider <select id="payment-provider"><option value="stripe">Stripe</option><option value="iyzico">iyzico</option><option value="paytr">PayTR</option></select></label><button id="payment-submit" type="submit">Start payment</button></form><p id="payment-status" role="status"></p><pre id="payment-token" hidden></pre><a href="/shop">Back to shop</a><script>(function(){const form=document.getElementById('payment-form');const button=document.getElementById('payment-submit');const status=document.getElementById('payment-status');const token=document.getElementById('payment-token');form.addEventListener('submit',async function(event){event.preventDefault();button.disabled=true;status.textContent='Starting payment…';token.hidden=true;try{const response=await fetch('/api/commerce/orders/%s/payment-intent',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':document.getElementById('csrf-token').value},body:JSON.stringify({provider:document.getElementById('payment-provider').value})});const data=await response.json();if(!response.ok)throw new Error(data.error||'Payment intent unavailable');status.textContent='Payment intent created. Continue with the selected provider.';token.textContent=data.client_token||data.id||'';token.hidden=!token.textContent;}catch(error){status.textContent=error.message||'Payment could not be started.';button.disabled=false;}})})();</script></body></html>`, template.HTMLEscapeString(order.ID), template.HTMLEscapeString(order.PaymentStatus), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), template.HTMLEscapeString(order.ID))
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Commerce catalog</title></head><body><h1>Commerce catalog</h1><form method="post" action="/admin/commerce/products"><input type="hidden" name="_csrf" value="` + template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())) + `"><input name="sku" placeholder="SKU" required><input name="title" placeholder="Title" required><input name="price_cents" type="number" min="0" required><input name="currency" value="TRY" maxlength="3" required><input name="stock" type="number" min="0" required><select name="status"><option value="draft">Draft</option><option value="published">Published</option></select><button type="submit">Create product</button></form><ul>`))
	for _, product := range products {
		_, _ = fmt.Fprintf(w, `<li>%s (%s) — %d %s — stock %d — %s — <a href="/admin/commerce/products/%s/variants">Variants</a></li>`, template.HTMLEscapeString(product.Title), template.HTMLEscapeString(product.SKU), product.PriceCents, template.HTMLEscapeString(product.Currency), product.Stock, template.HTMLEscapeString(string(product.Status)), template.HTMLEscapeString(product.ID))
	}
	_, _ = w.Write([]byte(`</ul></body></html>`))
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
	token := template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context()))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Product variants</title></head><body><a href="/admin/commerce/products">Catalog</a><h1>Product variants</h1><form method="post" action="/admin/commerce/products/%s/variants"><input type="hidden" name="_csrf" value="%s"><input name="sku" placeholder="SKU" required><input name="title" placeholder="Title" required><input name="price_cents" type="number" min="0" required><input name="stock" type="number" min="0" required><select name="status"><option value="draft">Draft</option><option value="published">Published</option></select><button type="submit">Create variant</button></form><ul>`, template.HTMLEscapeString(r.PathValue("productID")), token)
	for _, variant := range variants {
		_, _ = fmt.Fprintf(w, `<li>%s (%s) — %d %s — stock %d — %s</li>`, template.HTMLEscapeString(variant.Title), template.HTMLEscapeString(variant.SKU), variant.PriceCents, template.HTMLEscapeString(variant.Currency), variant.Stock, template.HTMLEscapeString(string(variant.Status)))
	}
	_, _ = w.Write([]byte(`</ul></body></html>`))
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
	product := commerce.Product{ID: id, SKU: r.FormValue("sku"), Title: r.FormValue("title"), PriceCents: price, Currency: r.FormValue("currency"), Stock: stock, Status: commerce.ProductStatus(r.FormValue("status"))}
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
	_, _ = w.Write([]byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Commerce orders</title></head><body><nav><a href="/admin/commerce/products">Catalog</a></nav><h1>Commerce orders</h1><table><thead><tr><th>Order</th><th>Customer</th><th>Total</th><th>Payment</th><th>Shipping</th><th>Items</th><th>Refund</th></tr></thead><tbody>`))
	for _, order := range orders {
		refund := `<span>Unavailable</span>`
		if order.PaymentStatus == "paid" {
			refund = fmt.Sprintf(`<form method="post" action="/admin/commerce/orders/%s/refund"><input type="hidden" name="_csrf" value="%s"><input name="provider" placeholder="stripe" required><input name="payment_id" placeholder="provider payment id" required><input name="amount_cents" type="number" min="1" value="%d" required><button type="submit">Refund full order</button></form>`, template.HTMLEscapeString(order.ID), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), order.TotalCents)
		}
		_, _ = fmt.Fprintf(w, `<tr><td><code>%s</code></td><td><code>%s</code></td><td>%d %s</td><td>%s</td><td>%s</td><td>%d</td><td>%s</td></tr>`, template.HTMLEscapeString(order.ID), template.HTMLEscapeString(order.UserID), order.TotalCents, template.HTMLEscapeString(order.Currency), template.HTMLEscapeString(order.PaymentStatus), template.HTMLEscapeString(order.ShippingStatus), len(order.Items), refund)
	}
	_, _ = w.Write([]byte(`</tbody></table></body></html>`))
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
	id, err := newID()
	if err != nil {
		http.Error(w, "checkout unavailable", http.StatusServiceUnavailable)
		return
	}
	order, err := h.Commerce.PlaceOrder(r.Context(), commerce.Checkout{ID: id, UserID: user.ID, IdempotencyKey: key, CartID: input.CartID, Items: input.Items})
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
