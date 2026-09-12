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
	"time"

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
	Limiter           security.RateLimiter
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
		{Pattern: "GET /api/commerce/products/{productID}", Handler: http.HandlerFunc(h.getProduct), Protected: false},
		{Pattern: "PUT /api/commerce/products/{productID}", Handler: http.HandlerFunc(h.updateProduct), Protected: true},
		{Pattern: "GET /api/commerce/products/{productID}/variants", Handler: http.HandlerFunc(h.variants), Protected: false},
		{Pattern: "POST /api/commerce/products/{productID}/variants", Handler: http.HandlerFunc(h.createVariant), Protected: true},
		{Pattern: "GET /api/commerce/cart", Handler: http.HandlerFunc(h.cart), Protected: true},
		{Pattern: "POST /api/commerce/cart/items", Handler: http.HandlerFunc(h.addCartItem), Protected: true},
		{Pattern: "DELETE /api/commerce/cart/items/{productID}", Handler: http.HandlerFunc(h.removeCartItem), Protected: true},
		{Pattern: "POST /api/commerce/cart/checkout", Handler: http.HandlerFunc(h.checkout), Protected: true},
		{Pattern: "POST /api/commerce/orders/{orderID}/payment-intent", Handler: http.HandlerFunc(h.paymentIntent), Protected: true},
		{Pattern: "GET /shop", Handler: http.HandlerFunc(h.shop), Protected: false},
		{Pattern: "GET /shop/products/{productID}", Handler: http.HandlerFunc(h.productDetailPage), Protected: false},
		{Pattern: "GET /shop/products/{productID}/html", Handler: http.HandlerFunc(h.productDetailPage), Protected: false},
		{Pattern: "GET /shop/cart", Handler: http.HandlerFunc(h.cartPage), Protected: false},
		{Pattern: "POST /shop/cart/items", Handler: http.HandlerFunc(h.addCartItemForm), Protected: false},
		{Pattern: "POST /shop/cart/items/{productID}/remove", Handler: http.HandlerFunc(h.removeCartItemForm), Protected: false},
		{Pattern: "GET /shop/checkout", Handler: http.HandlerFunc(h.checkoutPage), Protected: false},
		{Pattern: "POST /shop/checkout", Handler: http.HandlerFunc(h.checkoutForm), Protected: false},
		{Pattern: "GET /admin/commerce/products", Handler: http.HandlerFunc(h.adminProducts), Protected: true},
		{Pattern: "GET /admin/commerce/products/new", Handler: http.HandlerFunc(h.adminNewProduct), Protected: true},
		{Pattern: "POST /admin/commerce/products", Handler: http.HandlerFunc(h.adminCreateProduct), Protected: true},
		{Pattern: "GET /admin/commerce/products/{productID}", Handler: http.HandlerFunc(h.adminProductPage), Protected: true},
		{Pattern: "POST /admin/commerce/products/{productID}", Handler: http.HandlerFunc(h.adminUpdateProduct), Protected: true},
		{Pattern: "GET /admin/commerce/products/{productID}/variants", Handler: http.HandlerFunc(h.adminVariants), Protected: true},
		{Pattern: "POST /admin/commerce/products/{productID}/variants", Handler: http.HandlerFunc(h.adminCreateVariant), Protected: true},
		{Pattern: "GET /admin/commerce/orders", Handler: http.HandlerFunc(h.adminOrders), Protected: true},
		{Pattern: "POST /admin/commerce/orders/{orderID}/refund", Handler: http.HandlerFunc(h.adminRefund), Protected: true},
	}
}

type productInput struct {
	SKU         string                 `json:"sku"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	MediaID     string                 `json:"media_id"`
	PriceCents  int64                  `json:"price_cents"`
	Currency    string                 `json:"currency"`
	Stock       int                    `json:"stock"`
	Status      commerce.ProductStatus `json:"status"`
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
	if h.Limiter != nil && !h.Limiter.Allow("payment-intent:"+user.ID+":"+requestClientIP(r)) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
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
	product := commerce.Product{ID: id, SKU: input.SKU, Title: input.Title, Description: input.Description, MediaID: input.MediaID, PriceCents: input.PriceCents, Currency: input.Currency, Stock: input.Stock, Status: input.Status}
	if err := h.Catalog.CreateProduct(r.Context(), product); err != nil {
		http.Error(w, "invalid product", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(product)
}

func (h Handler) getProduct(w http.ResponseWriter, r *http.Request) {
	product, err := h.Catalog.GetProduct(r.Context(), h.DB, r.PathValue("productID"))
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(product)
}

func (h Handler) updateProduct(w http.ResponseWriter, r *http.Request) {
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
	productID := r.PathValue("productID")
	product := commerce.Product{ID: productID, SKU: input.SKU, Title: input.Title, Description: input.Description, MediaID: input.MediaID, PriceCents: input.PriceCents, Currency: input.Currency, Stock: input.Stock, Status: input.Status}
	if err := h.Catalog.UpdateProduct(r.Context(), product); err != nil {
		http.Error(w, "invalid product", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(product)
}

func (h Handler) productDetailPage(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("productID")
	product, err := h.Catalog.GetProduct(r.Context(), h.DB, productID)
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}
	variants := []commerce.ProductVariant(nil)
	if h.CatalogVariants != nil {
		variants, _ = h.Catalog.ListVariants(r.Context(), h.DB, product.ID, true)
	}

	user := auth.UserFromContext(r.Context())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, commerceDocumentStart("HydroGo Mağaza · "+product.Title))

	var variantOptions strings.Builder
	if len(variants) > 0 {
		variantOptions.WriteString(`<label class="variant-select-label">Seçenek / Model <select name="variant_id" required><option value="">Lütfen model seçin…</option>`)
		for _, v := range variants {
			if v.Status == commerce.ProductOutOfStock || v.Stock == 0 {
				continue
			}
			variantOptions.WriteString(fmt.Sprintf(`<option value="%s">%s — %.2f %s (Stok: %d)</option>`, template.HTMLEscapeString(v.ID), template.HTMLEscapeString(v.Title), float64(v.PriceCents)/100.0, template.HTMLEscapeString(v.Currency), v.Stock))
		}
		variantOptions.WriteString(`</select></label>`)
	}

	buyButton := `<button class="btn-primary" type="submit" style="width:100%;justify-content:center;padding:15px;font-size:1.05rem;box-shadow:var(--theme-shadow-primary);">Sepete Ekle 🛍️</button>`
	if user == nil || !user.Active {
		buyButton = `<button class="btn-secondary" type="button" disabled title="Sepete eklemek için giriş gereklidir" style="width:100%;justify-content:center;padding:15px;font-size:1.02rem;opacity:.65;">Sepete Eklemek İçin Giriş Yapın</button>`
	}

	imgMarkup := productMediaMarkup(product)
	if imgMarkup == "" {
		imgMarkup = `<div class="product-gallery-placeholder">📦</div>`
	}

	// Related products
	allProducts, _ := h.Catalog.ListProducts(r.Context(), h.DB, true)
	var relatedBuilder strings.Builder
	var relatedCount int
	for _, other := range allProducts {
		if other.ID == product.ID {
			continue
		}
		if relatedCount == 0 {
			relatedBuilder.WriteString(`<section class="related-products-section"><div class="related-section-header"><div><div class="section-badge">ÖNERİLENLER</div><h2>İlginizi Çekebilecek Diğer Ürünler</h2></div><a href="/shop" style="font-weight:700;font-size:0.92rem;">Tümünü Gör →</a></div><div class="shop-grid">`)
		}
		relImg := productMediaMarkup(other)
		if relImg == "" {
			relImg = `<div style="height:180px;background:#f8fafc;display:flex;align-items:center;justify-content:center;font-size:3rem;border-bottom:1px solid var(--theme-line);">📦</div>`
		}
		relatedBuilder.WriteString(fmt.Sprintf(`<article class="product-card">%s<div class="product-card-body"><h2><a href="/shop/products/%s">%s</a></h2><div class="product-description">%s</div><div class="product-pricing"><span class="product-price">%.2f %s</span><span class="product-stock-badge">Stok: %d</span></div><a class="product-card-link btn-primary" href="/shop/products/%s">Ürünü İncele <span aria-hidden="true">→</span></a></div></article>`,
			relImg,
			template.HTMLEscapeString(other.ID),
			template.HTMLEscapeString(other.Title),
			ui.RichTextHTML(other.Description),
			float64(other.PriceCents)/100.0,
			template.HTMLEscapeString(other.Currency),
			other.Stock,
			template.HTMLEscapeString(other.ID),
		))
		relatedCount++
		if relatedCount >= 4 {
			break
		}
	}
	if relatedCount > 0 {
		relatedBuilder.WriteString(`</div></section>`)
	}

	_, _ = fmt.Fprintf(w, `<main data-testid="public-product" class="news-portal"><div class="news-breadcrumb" style="margin-bottom:24px;display:flex;align-items:center;gap:8px;font-size:0.9rem;"><a href="/" style="color:var(--theme-muted);">Ana Sayfa</a><span style="color:#cbd5e1;">/</span><a href="/shop" style="color:var(--theme-muted);">Mağaza Kataloğu</a><span style="color:#cbd5e1;">/</span><span style="color:var(--theme-primary);font-weight:700;">%s</span></div><div class="product-detail-layout"><div class="product-main-col"><div class="product-gallery-card"><div class="product-gallery-pane">%s</div><div class="product-trust-grid"><div class="product-trust-item"><span class="product-trust-icon">⚡</span><div class="product-trust-text"><span class="product-trust-title">Hızlı Teslimat</span><span class="product-trust-desc">Dijital veya fiziksel teslimat seçenekleri</span></div></div><div class="product-trust-item"><span class="product-trust-icon">🛡️</span><div class="product-trust-text"><span class="product-trust-title">3D Secure Ödeme</span><span class="product-trust-desc">Güvenli ödeme sağlayıcılarıyla korumalı işlem</span></div></div><div class="product-trust-item"><span class="product-trust-icon">📦</span><div class="product-trust-text"><span class="product-trust-title">Orijinal Ürün &amp; Fatura</span><span class="product-trust-desc">Yapılandırmaya bağlı kurumsal fatura seçenekleri</span></div></div><div class="product-trust-item"><span class="product-trust-icon">🔄</span><div class="product-trust-text"><span class="product-trust-title">İade koşulları ürün politikasına göre</span><span class="product-trust-desc">Yayınlanan iade koşulları ve müşteri desteği</span></div></div></div></div><div class="product-tabs-card"><div class="product-section-block"><h3 class="product-section-heading"><span class="heading-icon">📋</span> Ürün Detayları ve Açıklama</h3><div class="product-description-rich">%s</div></div><div class="product-section-block"><h3 class="product-section-heading"><span class="heading-icon">⚙️</span> Teknik &amp; Ürün Özellikleri</h3><table class="product-specs-table"><tbody><tr><th>Ürün Kodu (SKU)</th><td><code style="font-family:'JetBrains Mono',monospace;background:var(--theme-primary-light);color:var(--theme-primary-dark);padding:2px 8px;border-radius:6px;font-size:0.85rem;">%s</code></td></tr><tr><th>Para Birimi / Fiyatlandırma</th><td>%s (Vergi uygulaması yapılandırmaya göre)</td></tr><tr><th>Stok Durumu</th><td>%d Adet Aktif Stok</td></tr><tr><th>Teslimat Türü</th><td>Anında Dijital Erişim / Güvenli Kargo</td></tr><tr><th>Lisanslama &amp; Destek</th><td>HydroGo Kurumsal Standart &amp; 1 Yıl Güncelleme</td></tr><tr><th>Ödeme Altyapıları</th><td>Stripe, iyzico, PayTR (Kredi Kartı, Banka Kartı, Havale/EFT)</td></tr></tbody></table></div><div class="product-section-block"><h3 class="product-section-heading"><span class="heading-icon">🚚</span> Teslimat, İade ve Değişim Koşulları</h3><p style="color:var(--theme-ink-subtle);font-size:0.92rem;line-height:1.6;margin:0 0 10px;">Siparişleriniz güvenli ödeme onayının ardından derhal işleme alınır. Dijital ve lisanslı ürünlerde erişim anahtarları hesabınıza ve e-posta adresinize anında iletilir.</p><p style="color:var(--theme-ink-subtle);font-size:0.92rem;line-height:1.6;margin:0;">Tüketici hakları kapsamında, teslim aldığınız fiziksel ürünleri 14 gün içerisinde herhangi bir gerekçe göstermeksizin iade edebilirsiniz.</p></div><div class="product-section-block"><h3 class="product-section-heading"><span class="heading-icon">❓</span> Sıkça Sorulan Sorular</h3><details class="product-faq-item" open><summary class="product-faq-summary">Siparişim ne zaman ve nasıl teslim edilir?</summary><div class="product-faq-body">Ödemeniz tamamlandığı anda sipariş onayınız oluşturulur. Dijital içerikler ve lisans kodları anında profilinize yansıtılır; fiziksel kargolar kargo süresi ürün politikasına göre belirlenir.</div></details><details class="product-faq-item"><summary class="product-faq-summary">Kurumsal fatura kesebilir misiniz?</summary><div class="product-faq-body">Evet. Sipariş tamamlama adımında şirket unvanı ve vergi numaranızı girdiğiniz takdirde adınıza veya şirketinize e-Arşiv / e-Fatura düzenlenir.</div></details><details class="product-faq-item"><summary class="product-faq-summary">Ödeme adımı güvenli mi?</summary><div class="product-faq-body">HydroGo Commerce modülü 3D Secure ve yapılandırılmış ödeme sağlayıcıları üzerinden güvenli ödeme akışı sağlar.</div></details></div></div></div><div class="product-side-col"><div class="product-sticky-box"><div class="product-kicker-bar"><span class="product-sku">%s</span><div class="product-rating-badge"><span class="product-rating-stars">★★★★★</span><span>Ürün bilgileri</span></div></div><h1>%s</h1><div class="product-price-panel"><div class="product-price-main"><span class="product-price-value">%.2f %s</span><span class="product-vat-tag">Vergi uygulaması yapılandırmaya göre</span></div><div class="product-stock-indicator"><span class="pulse-green-dot"></span><span>%d Adet Stokta</span></div></div><form class="product-order-form" method="post" action="/shop/cart/items" style="display:grid;gap:18px;"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="product_id" value="%s">%s<div class="qty-btn-group" style="display:flex;align-items:center;justify-content:space-between;background:var(--theme-soft);border:1px solid var(--theme-line);padding:10px 14px;border-radius:var(--theme-radius-md);"><label style="display:flex;align-items:center;gap:10px;font-weight:700;font-size:0.9rem;color:var(--theme-ink);"><span>Sipariş Adedi:</span></label><input name="quantity" type="number" min="1" max="1000" value="1" style="width:75px;padding:6px 10px;border:1px solid #cbd5e1;border-radius:8px;font-weight:700;text-align:center;"></div>%s</form><ul class="product-perks-checklist"><li><span class="perk-check">✓</span> Anında onay ve sipariş takibi</li><li><span class="perk-check">✓</span> 3D Secure güvenli ödeme koruması</li><li><span class="perk-check">✓</span> Kurumsal e-fatura &amp; muhasebe uyumlu</li><li><span class="perk-check">✓</span> Yayınlanan destek seçenekleri</li></ul><div class="product-support-box"><span style="font-size:1.2rem;">💬</span><div><strong>Destek mi lazım?</strong><div style="font-size:0.78rem;color:#3b82f6;">Sorularınız için bizimle iletişime geçin.</div></div></div></div></div></div>%s</main>%s`,
		template.HTMLEscapeString(product.Title),
		imgMarkup,
		ui.RichTextHTML(product.Description),
		template.HTMLEscapeString(product.SKU),
		template.HTMLEscapeString(product.Currency),
		product.Stock,
		template.HTMLEscapeString(product.SKU),
		template.HTMLEscapeString(product.Title),
		float64(product.PriceCents)/100.0,
		template.HTMLEscapeString(product.Currency),
		product.Stock,
		template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())),
		template.HTMLEscapeString(product.ID),
		variantOptions.String(),
		buyButton,
		relatedBuilder.String(),
		commerceEnterpriseFooter(),
	)
}

func (h Handler) shop(w http.ResponseWriter, r *http.Request) {
	products, err := h.Catalog.ListProducts(r.Context(), h.DB, true)
	if err != nil {
		http.Error(w, "shop unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	user := auth.UserFromContext(r.Context())
	_, _ = fmt.Fprint(w, commerceDocumentStart("HydroGo Mağaza · E-Ticaret Kataloğu"))
	_, _ = fmt.Fprintf(w, `<main class="news-portal"><section class="news-hero-banner"><div class="news-hero-content"><div class="news-tag-badge">HYDROGO COMMERCE</div><h1>E-Ticaret &amp; Ürün Kataloğu</h1><p>Panelden yönetilen ürünler, çoklu varyantlar, dinamik fiyatlandırma ve stok rezervasyonu ile uçtan uca modern alışveriş akışı.</p><div class="news-hero-actions"><a class="btn-primary" href="/shop/cart">Sepeti Görüntüle →</a><a class="btn-secondary" href="/admin/commerce/products">Ürünleri Panelde Yönet ↗</a></div></div></section><span data-testid="public-shop" class="sr-only">Mağaza</span>`)
	if user == nil || !user.Active {
		_, _ = w.Write([]byte(`<aside class="login-notice" style="background:#eff6ff;border:1px solid #bfdbfe;border-radius:12px;padding:16px 20px;margin-bottom:32px;color:#1e40af;"><strong>Sepete eklemek ve sipariş vermek için giriş yapın:</strong> <a href="/login?return_to=%2Fshop" style="color:var(--theme-primary);font-weight:700;text-decoration:underline;">Giriş Yap</a> veya <a href="/admin" style="color:var(--theme-primary);font-weight:700;text-decoration:underline;">Yönetici Hesabı</a></aside>`))
	}
	_, _ = fmt.Fprintf(w, `<section class="news-grid-section"><div class="section-header" style="margin-bottom:24px;"><div><div class="section-badge">KATALOG</div><h2 style="font-size:1.8rem;font-weight:800;color:var(--theme-ink);margin:4px 0 0;">Tüm Ürünler (%d)</h2></div><span class="section-desc" style="color:var(--theme-muted);font-size:.95rem;">Aktif stok ve varyant seçenekleri</span></div><div class="shop-grid">`, len(products))
	for _, product := range products {
		img := productMediaMarkup(product)
		if img == "" {
			img = `<div style="height:180px;background:#f8fafc;display:flex;align-items:center;justify-content:center;font-size:3rem;border-bottom:1px solid var(--theme-line);">📦</div>`
		}
		_, _ = fmt.Fprintf(w, `<article class="product-card">%s<div class="product-card-body"><h2><a href="/shop/products/%s">%s</a></h2><div class="product-description">%s</div><div class="product-pricing"><span class="product-price">%.2f %s</span><span class="product-stock-badge">Stok: %d</span></div><a class="product-card-link btn-primary" href="/shop/products/%s">Ürünü incele <span aria-hidden="true">→</span></a></div></article>`, img, template.HTMLEscapeString(product.ID), template.HTMLEscapeString(product.Title), ui.RichTextHTML(product.Description), float64(product.PriceCents)/100.0, template.HTMLEscapeString(product.Currency), product.Stock, template.HTMLEscapeString(product.ID))
	}
	_, _ = fmt.Fprintf(w, `</div></section></main>%s`, commerceEnterpriseFooter())
}

const cartCookieName = "hydro_cart_session"

func (h Handler) resolveCartUserID(w http.ResponseWriter, r *http.Request) string {
	if user := auth.UserFromContext(r.Context()); user != nil && user.Active && strings.TrimSpace(user.ID) != "" {
		return user.ID
	}
	if cookie, err := r.Cookie(cartCookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return cookie.Value
	}
	if cookie, err := r.Cookie("hydro_member_session"); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return "member_" + cookie.Value
	}
	id, err := newID()
	if err != nil {
		id = fmt.Sprintf("guest-%d", time.Now().UnixNano())
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cartCookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 30,
	})
	return id
}

func (h Handler) addCartItemForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid cart item", http.StatusBadRequest)
		return
	}
	quantity, err := strconv.Atoi(r.FormValue("quantity"))
	if err != nil || quantity < 1 {
		quantity = 1
	}
	userID := h.resolveCartUserID(w, r)
	if _, err := h.Cart.Add(r.Context(), userID, r.FormValue("product_id"), r.FormValue("variant_id"), quantity); err != nil {
		http.Error(w, "cannot add cart item", http.StatusConflict)
		return
	}
	http.Redirect(w, r, "/shop/cart", http.StatusSeeOther)
}

func (h Handler) cartPage(w http.ResponseWriter, r *http.Request) {
	userID := h.resolveCartUserID(w, r)
	cart, err := h.Cart.Get(r.Context(), h.DB, userID)
	if err != nil {
		http.Error(w, "cart unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, commerceDocumentStart("Alışveriş Sepetim · HydroGo"))
	_, _ = fmt.Fprintf(w, `<main class="news-portal cart-page"><div class="cart-shell"><div class="cart-header-row"><div class="section-badge"><span class="badge-dot"></span> HYDROGO COMMERCE</div><h1 class="cart-title">Alışveriş Sepetim</h1><p class="cart-subtitle">Seçtiğiniz ürünler, anında lisans teslimatı ve güvenli sipariş adımı.</p></div>`)
	if len(cart.Items) == 0 {
		_, _ = w.Write([]byte(`<div class="cart-empty-card"><div class="cart-empty-icon"><svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="21" r="1"></circle><circle cx="20" cy="21" r="1"></circle><path d="M1 1h4l2.68 13.39a2 2 0 0 0 2 1.61h9.72a2 2 0 0 0 2-1.61L23 6H6"></path></svg></div><h2>Sepetiniz Şu Anda Boş</h2><p>Mağazamızdaki kurumsal yazılımları, dijital lisansları ve ürünleri inceleyip sepetinize ekleyebilirsiniz.</p><a class="btn-primary auth-submit-btn" href="/shop" style="display:inline-flex;width:auto;padding:12px 28px;"><span>Mağazayı Keşfet →</span></a></div>`))
	} else {
		_, _ = w.Write([]byte(`<div class="cart-content-grid"><div class="cart-items-column"><ul class="cart-list">`))
		for _, item := range cart.Items {
			_, _ = fmt.Fprintf(w, `<li class="cart-item-card"><div class="cart-item-icon-box"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line></svg></div><div class="cart-item-info"><h3>%s</h3><div class="cart-item-meta"><span class="cart-item-unit">%d adet × %.2f %s</span><span class="cart-item-sku">SKU: %s</span></div></div><div class="cart-item-end"><strong class="cart-item-price">%.2f %s</strong><form method="post" action="/shop/cart/items/%s/remove"><input type="hidden" name="_csrf" value="%s"><button type="submit" class="cart-item-remove-btn" title="Sepetten Kaldır"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg> <span>Kaldır</span></button></form></div></li>`, template.HTMLEscapeString(item.Title), item.Quantity, float64(item.UnitCents)/100.0, template.HTMLEscapeString(cart.Currency), template.HTMLEscapeString(item.SKU), float64(item.TotalCents)/100.0, template.HTMLEscapeString(cart.Currency), template.HTMLEscapeString(item.ProductID), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())))
		}
		_, _ = fmt.Fprintf(w, `</ul></div><div class="cart-summary-column"><div class="cart-summary-card"><h3 class="cart-summary-heading">Sipariş Özeti</h3><div class="cart-summary-row"><span>Ara Toplam</span><strong>%.2f %s</strong></div><div class="cart-summary-row"><span>KDV (Dahil)</span><span>%s</span></div><div class="cart-summary-divider"></div><div class="cart-summary-total"><span>Genel Toplam</span><strong>%.2f %s</strong></div><a class="btn-primary auth-submit-btn cart-checkout-btn" href="/shop/checkout"><span>Siparişi Tamamla &amp; Ödeme →</span></a><div class="auth-trust-badge" style="margin-top:16px;padding-top:14px;"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg><span>Ödeme sağlayıcısının güvenlik politikaları geçerlidir</span></div></div></div></div>`, float64(cart.TotalCents)/100.0, template.HTMLEscapeString(cart.Currency), "%20", float64(cart.TotalCents)/100.0, template.HTMLEscapeString(cart.Currency))
	}
	_, _ = fmt.Fprintf(w, `</div></main>%s`, commerceEnterpriseFooter())
}

func (h Handler) checkoutPage(w http.ResponseWriter, r *http.Request) {
	userID := h.resolveCartUserID(w, r)
	cart, err := h.Cart.Get(r.Context(), h.DB, userID)
	if err != nil {
		http.Error(w, "cart unavailable", http.StatusServiceUnavailable)
		return
	}
	if len(cart.Items) == 0 {
		http.Redirect(w, r, "/shop/cart", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, commerceDocumentStart("Ödeme ve Sipariş Onayı · HydroGo"))
	_, _ = fmt.Fprintf(w, `<main class="news-portal cart-page" style="max-width:780px;"><div class="cart-shell"><div class="checkout-card"><div class="section-badge"><span class="badge-dot"></span> GÜVENLİ SİPARİŞ ADIMI</div><h1 class="cart-title">Teslimat &amp; Sipariş Bilgileri</h1><p class="cart-subtitle">Toplam Ödeme Tutarı: <strong style="color:#38bdf8;font-size:1.25rem;">%.2f %s</strong> (Vergi uygulaması yapılandırmaya göre)</p><form class="member-auth-form" method="post" action="/shop/checkout" style="margin-top:24px;"><input type="hidden" name="_csrf" value="%s"><div style="display:grid;grid-template-columns:1fr 1fr;gap:16px;"><label class="auth-field"><span class="field-label">Ad Soyad</span><div class="field-input-wrap"><input name="customer_name" required maxlength="120" placeholder="Adınız Soyadınız"></div></label><label class="auth-field"><span class="field-label">Vergi / fatura bilgisi (gerekiyorsa)</span><div class="field-input-wrap"><input name="customer_identity_number" maxlength="20" placeholder="Vergi numarası (opsiyonel)"></div></label></div><div style="display:grid;grid-template-columns:1fr 1fr;gap:16px;"><label class="auth-field"><span class="field-label">E-Posta Adresi</span><div class="field-input-wrap"><input name="customer_email" type="email" required maxlength="254" placeholder="musteri@example.com"></div></label><label class="auth-field"><span class="field-label">Telefon Numarası</span><div class="field-input-wrap"><input name="customer_phone" required maxlength="32" placeholder="+90 555 000 0000"></div></label></div><label class="auth-field"><span class="field-label">Teslimat &amp; Fatura Adresi</span><div class="field-input-wrap"><textarea name="customer_address" class="cart-textarea" required maxlength="500" rows="3" placeholder="Açık adres, sokak, mahalle, bina ve daire no"></textarea></div></label><div style="display:grid;grid-template-columns:2fr 1fr 1fr;gap:16px;"><label class="auth-field"><span class="field-label">Şehir</span><div class="field-input-wrap"><input name="customer_city" required maxlength="100" placeholder="İstanbul"></div></label><label class="auth-field"><span class="field-label">Ülke</span><div class="field-input-wrap"><input name="customer_country" value="TR" required maxlength="2"></div></label><label class="auth-field"><span class="field-label">Posta Kodu</span><div class="field-input-wrap"><input name="customer_postal_code" maxlength="20" placeholder="34000"></div></label></div><button class="btn-primary auth-submit-btn" type="submit" style="margin-top:12px;padding:15px 24px;font-size:1.02rem;"><span>Siparişi Oluştur ve Ödemeye Geç 💳</span></button></form><div class="auth-trust-badge"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path></svg><span>Yapılandırılmış ödeme sağlayıcısı üzerinden ödeme</span></div></div></div></main>%s`, float64(cart.TotalCents)/100.0, template.HTMLEscapeString(cart.Currency), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), commerceEnterpriseFooter())
}

func commerceDocumentStart(title string) string {
	return fmt.Sprintf(`<!doctype html><html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>%s</title><link rel="stylesheet" href="/theme/static/theme.css?v=20260912clean"></head><body class="theme-corporate-body"><header class="news-header"><div class="header-container"><a class="news-logo" href="/" aria-label="HydroGo Ana Sayfa"><div class="logo-mark"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg></div><div class="logo-text"><span>HYDROGO</span></div></a><button class="nav-toggle" type="button" aria-expanded="false" aria-controls="primary-nav" aria-label="Menüyü aç"><span class="hamburger-bar"></span><span class="hamburger-bar"></span><span class="hamburger-bar"></span></button><nav id="primary-nav" class="news-nav" aria-label="Ana navigasyon"><a href="/" class="nav-item">Ana Sayfa</a><a href="/public/pages/tr/showcase/html" class="nav-item">Showcase</a><a href="/public/pages/tr/about/html" class="nav-item">Hakkında</a><a href="/public/news/tr/html" class="nav-item">Haberler</a><a href="/shop" class="nav-item is-active">Mağaza</a><a href="/shop/cart" class="nav-item">Sepetim</a><a href="/member/login" class="nav-item nav-login-link">Giriş</a><a class="nav-admin-btn" href="/admin">Studio Paneli ↗</a></nav></div></header>`, template.HTMLEscapeString(title))
}

func commerceEnterpriseFooter() string {
	return `<footer class="news-footer"><div class="footer-container"><div class="footer-brand-col"><div class="footer-logo"><div class="logo-mark"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg></div><strong>HYDROGO <span>PLATFORM</span></strong></div><p class="footer-summary">Yüksek performanslı Go altyapısı, katı mimari sınırlar ve sözleşme testli çift veritabanı desteği ile kurumsal içerik ve e-ticaret platformu.</p><div class="footer-badge-list"><span class="f-badge">⚡ Go 1.22+</span><span class="f-badge">🛡️ Tip Güvenli</span><span class="f-badge">📦 Modüler Core</span></div></div><div class="footer-nav-col"><h4>Platform</h4><nav><a href="/">Ana Sayfa</a><a href="/public/pages/tr/showcase/html">Showcase</a><a href="/public/pages/tr/about/html">Hakkında</a><a href="/public/news/tr/html">Haberler & Blog</a></nav></div><div class="footer-nav-col"><h4>Çözümler</h4><nav><a href="/shop">E-Ticaret Mağazası</a><a href="/admin">Admin Studio Paneli</a><a href="/api/v1/pages">REST API v1</a></nav></div></div><div class="footer-bottom-bar"><p>© 2026 HydroGo Platform. Tüm hakları saklıdır.</p><div class="footer-sub-links"><a href="/public/pages/tr/about/html">Güvenlik & Felsefe</a><span>•</span><a href="/admin">Yönetim Girişi</a></div></div></footer><script src="/theme/static/theme.js?v=20260912"></script></body></html>`
}

func (h Handler) checkoutForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid checkout", http.StatusBadRequest)
		return
	}
	userID := h.resolveCartUserID(w, r)
	cart, err := h.Cart.Get(r.Context(), h.DB, userID)
	if err != nil || cart.ID == "" || len(cart.Items) == 0 {
		http.Error(w, "cart is empty", http.StatusConflict)
		return
	}
	orderID, err := newID()
	if err != nil {
		http.Error(w, "checkout unavailable", http.StatusServiceUnavailable)
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		key, err = newID()
		if err != nil {
			http.Error(w, "checkout unavailable", http.StatusServiceUnavailable)
			return
		}
	}
	if len(key) > 128 {
		http.Error(w, "invalid idempotency key", http.StatusBadRequest)
		return
	}
	order, err := h.Commerce.PlaceOrder(r.Context(), commerce.Checkout{ID: orderID, UserID: userID, CartID: cart.ID, IdempotencyKey: key, Customer: commerce.PaymentCustomer{Name: r.FormValue("customer_name"), IdentityNumber: r.FormValue("customer_identity_number"), Email: r.FormValue("customer_email"), Phone: r.FormValue("customer_phone"), Address: r.FormValue("customer_address"), City: r.FormValue("customer_city"), Country: r.FormValue("customer_country"), PostalCode: r.FormValue("customer_postal_code"), ClientIP: requestClientIP(r)}})
	if err != nil {
		http.Error(w, "checkout failed", http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, commerceDocumentStart("Sipariş Alındı · HydroGo"))
	orderHTML := `<main class="news-portal cart-page" style="max-width:780px;"><div class="cart-shell"><div class="checkout-card"><div class="section-badge" style="background:rgba(16,185,129,0.12);border-color:rgba(16,185,129,0.3);color:#34d399;"><span class="badge-dot" style="background:#34d399;box-shadow:0 0 8px #34d399;"></span> SİPARİŞ ONAYLANDI</div><h1 class="cart-title">Siparişiniz Başarıyla Alındı</h1><p class="cart-subtitle">Sipariş Kodu: <code style="font-family:var(--font-mono);background:rgba(6,182,212,0.12);color:#38bdf8;padding:4px 10px;border-radius:8px;font-size:.92rem;border:1px solid rgba(6,182,212,0.3);">` + template.HTMLEscapeString(order.ID) + `</code> &bull; Durum: <strong style="color:#34d399;">` + template.HTMLEscapeString(order.PaymentStatus) + `</strong></p><div style="background:linear-gradient(165deg,rgba(19,28,46,0.9) 0%,rgba(11,15,25,0.95) 100%);border:1px solid rgba(96,165,250,0.25);border-radius:20px;padding:28px 24px;margin:28px 0 24px;"><h3 style="margin:0 0 10px;font-size:1.15rem;color:#fff;font-weight:700;">Ödeme Başlatma</h3><p style="margin:0 0 18px;color:var(--smoke);font-size:.92rem;">Ödeme sağlayıcınızı seçip 3D Secure provizyon adımına geçebilirsiniz:</p><form id="payment-form" style="display:grid;gap:16px;"><input type="hidden" id="csrf-token" value="` + template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())) + `"><label class="auth-field"><span class="field-label">Ödeme Sağlayıcı Altyapısı</span><div class="field-input-wrap"><select id="payment-provider" class="cart-provider-select"><option value="stripe">Stripe Checkout</option><option value="iyzico">iyzico Güvenli Ödeme</option><option value="paytr">PayTR Direct Token</option></select></div></label><button id="payment-submit" class="btn-primary auth-submit-btn" type="submit"><span>Ödemeyi Tamamla 💳</span></button></form><p id="payment-status" role="status" style="margin:16px 0 0;font-weight:600;color:#38bdf8;"></p><pre id="payment-token" hidden style="margin-top:14px;padding:14px;background:#090d16;color:#38bdf8;border-radius:12px;font:11px/1.45 var(--font-mono);overflow-x:auto;border:1px solid rgba(6,182,212,0.25);"></pre></div><a class="auth-back-link" href="/shop">← Alışverişe Geri Dön</a></div></div></main>`
	_, _ = fmt.Fprint(w, orderHTML)
	scriptHTML := `<script nonce="` + template.HTMLEscapeString(security.NonceFromContext(r.Context())) + `">(function(){const form=document.getElementById('payment-form');const button=document.getElementById('payment-submit');const status=document.getElementById('payment-status');const token=document.getElementById('payment-token');if(!form)return;form.addEventListener('submit',async function(event){event.preventDefault();button.disabled=true;status.textContent='Ödeme başlatılıyor…';token.hidden=true;try{const response=await fetch('/api/commerce/orders/` + template.HTMLEscapeString(order.ID) + `/payment-intent',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':document.getElementById('csrf-token').value},body:JSON.stringify({provider:document.getElementById('payment-provider').value})});const data=await response.json().catch(function(){return {};});if(!response.ok)throw new Error(data.error||'Ödeme başlatılamadı');if(data.redirect_url){window.location.assign(data.redirect_url);return;}status.textContent='Ödeme talebi hazırlandı. Seçilen sağlayıcı ile devam ediliyor.';token.textContent=data.client_token||data.id||'';token.hidden=!token.textContent;}catch(error){status.textContent=error.message||'Ödeme başlatılamadı.';button.disabled=false;}})})();</script>`
	_, _ = fmt.Fprint(w, scriptHTML)
	_, _ = fmt.Fprint(w, commerceEnterpriseFooter())
}

func productMediaMarkup(product commerce.Product) template.HTML {
	if strings.TrimSpace(product.MediaID) == "" {
		return ""
	}
	return template.HTML(`<img class="product-image" src="/media/` + template.URLQueryEscaper(product.MediaID) + `?variant=card" alt="` + template.HTMLEscapeString(product.Title) + `" loading="lazy" data-fallback-src="/media/` + template.URLQueryEscaper(product.MediaID) + `">`)
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
	if err := ui.AdminStart(w, "Ürün Kataloğu", "", security.NonceFromContext(r.Context()), "commerce"); err != nil {
		return
	}
	_, _ = fmt.Fprint(w, `<section class="studio-card"><div class="table-wrap">`)
	rows := make([]adminstudio.Row, 0, len(products))
	for _, product := range products {
		rows = append(rows, adminstudio.Row{
			Values: map[string]string{
				"title":       product.Title,
				"sku":         product.SKU,
				"price_cents": strconv.FormatInt(product.PriceCents, 10),
				"currency":    product.Currency,
				"stock":       strconv.Itoa(product.Stock),
				"status":      string(product.Status),
			},
			Actions: []adminstudio.Action{
				{Label: "Düzenle", Href: "/admin/commerce/products/" + product.ID},
				{Label: "Varyantlar", Href: "/admin/commerce/products/" + product.ID + "/variants"},
			},
		})
	}
	if err := adminstudio.RenderList(w, adminstudio.ListData{Collection: collection, Rows: rows}); err != nil {
		http.Error(w, "admin studio catalog unavailable", http.StatusInternalServerError)
		return
	}
	_, _ = w.Write([]byte(`</div></section><script nonce="` + template.HTMLEscapeString(security.NonceFromContext(r.Context())) + `">(()=>{const sec=document.getElementById('new-product-section');const toggleBtn=document.getElementById('toggle-product-form-btn');if(toggleBtn&&sec){toggleBtn.addEventListener('click',()=>{const isHidden=sec.hasAttribute('hidden');if(isHidden){sec.removeAttribute('hidden');sec.scrollIntoView({behavior:'smooth'});}else{sec.setAttribute('hidden','');}});}if(window.location.hash==='#new-product'&&sec){sec.removeAttribute('hidden');sec.scrollIntoView({behavior:'smooth'});}})();</script>`))
	_ = ui.AdminEnd(w)
}

func (h Handler) adminNewProduct(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var form bytes.Buffer
	if err := adminstudio.RenderCreateForm(&form, adminstudio.FormData{
		Collection:  h.AdminCollection,
		Action:      "/admin/commerce/products",
		CSRFToken:   security.CSRFTokenFromContext(r.Context()),
		Nonce:       security.NonceFromContext(r.Context()),
		Values:      map[string]string{"currency": "TRY", "status": "draft"},
		SubmitLabel: "Ürünü oluştur",
	}); err != nil {
		http.Error(w, "admin studio catalog unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Yeni ürün", "Commerce", security.NonceFromContext(r.Context()), "commerce"); err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, `<section class="studio-card"><form class="studio-form" method="post" action="/admin/commerce/products">%s</form></section>`, form.String())
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
	_, _ = fmt.Fprintf(w, `<section class="studio-card"><div class="studio-actions" style="margin-bottom:16px;"><a class="button button-quiet" href="/admin/commerce/products">← Kataloğa dön</a></div><h2>Yeni varyant ekle</h2><p class="muted">SKU, fiyat ve stok seçeneklerini yönetin.</p><form class="studio-form" method="post" action="/admin/commerce/products/%s/variants">%s</form></section><section class="studio-card"><h2>Varyant listesi</h2><div class="table-wrap">`, template.HTMLEscapeString(r.PathValue("productID")), form.String())
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
	product := commerce.Product{ID: id, SKU: r.FormValue("sku"), Title: r.FormValue("title"), Description: r.FormValue("description"), MediaID: r.FormValue("media_id"), PriceCents: price, Currency: r.FormValue("currency"), Stock: stock, Status: commerce.ProductStatus(r.FormValue("status"))}
	if err := h.Catalog.CreateProduct(r.Context(), product); err != nil {
		http.Error(w, "invalid product", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/commerce/products", http.StatusSeeOther)
}

func (h Handler) adminProductPage(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil || !user.Active || user.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	productID := r.PathValue("productID")
	product, err := h.Catalog.GetProduct(r.Context(), h.DB, productID)
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}
	var form bytes.Buffer
	if err := adminstudio.RenderCreateForm(&form, adminstudio.FormData{
		Collection: h.AdminCollection,
		Action:     "/admin/commerce/products/" + product.ID,
		CSRFToken:  security.CSRFTokenFromContext(r.Context()),
		Nonce:      security.NonceFromContext(r.Context()),
		Values: map[string]string{
			"title":       product.Title,
			"sku":         product.SKU,
			"media_id":    product.MediaID,
			"price_cents": strconv.FormatInt(product.PriceCents, 10),
			"currency":    product.Currency,
			"stock":       strconv.Itoa(product.Stock),
			"status":      string(product.Status),
			"description": product.Description,
		},
		SubmitLabel: "Ürünü kaydet",
	}); err != nil {
		http.Error(w, "admin studio catalog unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ui.AdminStart(w, "Ürün Düzenle", "Commerce", security.NonceFromContext(r.Context()), "commerce"); err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, `<section class="studio-card"><div class="studio-actions" style="margin-bottom:16px;"><a class="button button-quiet" href="/admin/commerce/products">← Kataloğa dön</a><a class="button" href="/admin/commerce/products/%s/variants">Varyantları yönet →</a></div><form class="studio-form" method="post" action="/admin/commerce/products/%s">%s</form></section>`, template.HTMLEscapeString(product.ID), template.HTMLEscapeString(product.ID), form.String())
	_ = ui.AdminEnd(w)
}

func (h Handler) adminUpdateProduct(w http.ResponseWriter, r *http.Request) {
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
	productID := r.PathValue("productID")
	product := commerce.Product{
		ID:          productID,
		SKU:         r.FormValue("sku"),
		Title:       r.FormValue("title"),
		Description: r.FormValue("description"),
		MediaID:     r.FormValue("media_id"),
		PriceCents:  price,
		Currency:    r.FormValue("currency"),
		Stock:       stock,
		Status:      commerce.ProductStatus(r.FormValue("status")),
	}
	if err := h.Catalog.UpdateProduct(r.Context(), product); err != nil {
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
	if err := ui.AdminStart(w, "Siparişler", "Commerce", security.NonceFromContext(r.Context()), "orders"); err != nil {
		return
	}
	_, _ = w.Write([]byte(`<section class="studio-card"><div class="table-wrap"><table><thead><tr><th>Sipariş No</th><th>Müşteri</th><th>Tutar</th><th>Ödeme Durumu</th><th>Kargo Durumu</th><th>Ürün Adedi</th><th style="text-align:right">İşlem</th></tr></thead><tbody>`))
	for _, order := range orders {
		refund := `<span>-</span>`
		if order.PaymentStatus == "paid" {
			refund = fmt.Sprintf(`<form method="post" action="/admin/commerce/orders/%s/refund"><input type="hidden" name="_csrf" value="%s"><input name="provider" placeholder="stripe" required style="width:70px;display:inline-block;padding:4px 8px;font-size:.78rem;"><input name="payment_id" placeholder="id" required style="width:70px;display:inline-block;padding:4px 8px;font-size:.78rem;"><input name="amount_cents" type="number" min="1" value="%d" required style="width:60px;display:inline-block;padding:4px 8px;font-size:.78rem;"><button type="submit" class="button button-quiet button-danger" style="padding:4px 10px;font-size:.78rem;">İade et</button></form>`, template.HTMLEscapeString(order.ID), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), order.TotalCents)
		}
		_, _ = fmt.Fprintf(w, `<tr><td><code>%s</code></td><td><code>%s</code></td><td><strong>%d %s</strong></td><td><span class="status">%s</span></td><td>%s</td><td>%d</td><td style="text-align:right">%s</td></tr>`, template.HTMLEscapeString(order.ID), template.HTMLEscapeString(order.UserID), order.TotalCents, template.HTMLEscapeString(order.Currency), template.HTMLEscapeString(order.PaymentStatus), template.HTMLEscapeString(order.ShippingStatus), len(order.Items), refund)
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
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid cart item", http.StatusBadRequest)
		return
	}
	userID := h.resolveCartUserID(w, r)
	if err := h.Cart.Remove(r.Context(), userID, r.PathValue("productID")); err != nil {
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
	if h.Limiter != nil && !h.Limiter.Allow("checkout:"+user.ID+":"+requestClientIP(r)) {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
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
