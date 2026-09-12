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
		{Pattern: "GET /shop/cart", Handler: http.HandlerFunc(h.cartPage), Protected: true},
		{Pattern: "POST /shop/cart/items", Handler: http.HandlerFunc(h.addCartItemForm), Protected: true},
		{Pattern: "POST /shop/cart/items/{productID}/remove", Handler: http.HandlerFunc(h.removeCartItemForm), Protected: true},
		{Pattern: "GET /shop/checkout", Handler: http.HandlerFunc(h.checkoutPage), Protected: true},
		{Pattern: "POST /shop/checkout", Handler: http.HandlerFunc(h.checkoutForm), Protected: true},
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
	if h.Limiter != nil && !h.Limiter.Allow("payment-intent:"+user.ID) {
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

	_, _ = fmt.Fprintf(w, `<main class="news-portal"><div class="news-breadcrumb" style="margin-bottom:24px;display:flex;align-items:center;gap:8px;font-size:0.9rem;"><a href="/public/pages/tr/demo/html" style="color:var(--theme-muted);">Ana Sayfa</a><span style="color:#cbd5e1;">/</span><a href="/shop" style="color:var(--theme-muted);">Mağaza Kataloğu</a><span style="color:#cbd5e1;">/</span><span style="color:var(--theme-primary);font-weight:700;">%s</span></div><div class="product-detail-layout"><div class="product-main-col"><div class="product-gallery-card"><div class="product-gallery-pane">%s</div><div class="product-trust-grid"><div class="product-trust-item"><span class="product-trust-icon">⚡</span><div class="product-trust-text"><span class="product-trust-title">Hızlı Teslimat</span><span class="product-trust-desc">Anında dijital lisans veya 24 saatte kargoda</span></div></div><div class="product-trust-item"><span class="product-trust-icon">🛡️</span><div class="product-trust-text"><span class="product-trust-title">3D Secure Ödeme</span><span class="product-trust-desc">256-bit SSL ve uçtan uca şifreleme</span></div></div><div class="product-trust-item"><span class="product-trust-icon">📦</span><div class="product-trust-text"><span class="product-trust-title">Orijinal Ürün &amp; Fatura</span><span class="product-trust-desc">Resmi garantili, kurumsal e-fatura</span></div></div><div class="product-trust-item"><span class="product-trust-icon">🔄</span><div class="product-trust-text"><span class="product-trust-title">14 Gün İade Garantisi</span><span class="product-trust-desc">Koşulsuz iade ve müşteri desteği</span></div></div></div></div><div class="product-tabs-card"><div class="product-section-block"><h3 class="product-section-heading"><span class="heading-icon">📋</span> Ürün Detayları ve Açıklama</h3><div class="product-description-rich">%s</div></div><div class="product-section-block"><h3 class="product-section-heading"><span class="heading-icon">⚙️</span> Teknik &amp; Ürün Özellikleri</h3><table class="product-specs-table"><tbody><tr><th>Ürün Kodu (SKU)</th><td><code style="font-family:'JetBrains Mono',monospace;background:var(--theme-primary-light);color:var(--theme-primary-dark);padding:2px 8px;border-radius:6px;font-size:0.85rem;">%s</code></td></tr><tr><th>Para Birimi / Fiyatlandırma</th><td>%s (Tüm vergiler dahil)</td></tr><tr><th>Stok Durumu</th><td>%d Adet Aktif Stok</td></tr><tr><th>Teslimat Türü</th><td>Anında Dijital Erişim / Güvenli Kargo</td></tr><tr><th>Lisanslama &amp; Destek</th><td>HydroGo Kurumsal Standart &amp; 1 Yıl Güncelleme</td></tr><tr><th>Ödeme Altyapıları</th><td>Stripe, iyzico, PayTR (Kredi Kartı, Banka Kartı, Havale/EFT)</td></tr></tbody></table></div><div class="product-section-block"><h3 class="product-section-heading"><span class="heading-icon">🚚</span> Teslimat, İade ve Değişim Koşulları</h3><p style="color:var(--theme-ink-subtle);font-size:0.92rem;line-height:1.6;margin:0 0 10px;">Siparişleriniz güvenli ödeme onayının ardından derhal işleme alınır. Dijital ve lisanslı ürünlerde erişim anahtarları hesabınıza ve e-posta adresinize anında iletilir.</p><p style="color:var(--theme-ink-subtle);font-size:0.92rem;line-height:1.6;margin:0;">Tüketici hakları kapsamında, teslim aldığınız fiziksel ürünleri 14 gün içerisinde herhangi bir gerekçe göstermeksizin iade edebilirsiniz.</p></div><div class="product-section-block"><h3 class="product-section-heading"><span class="heading-icon">❓</span> Sıkça Sorulan Sorular</h3><details class="product-faq-item" open><summary class="product-faq-summary">Siparişim ne zaman ve nasıl teslim edilir?</summary><div class="product-faq-body">Ödemeniz tamamlandığı anda sipariş onayınız oluşturulur. Dijital içerikler ve lisans kodları anında profilinize yansıtılır; fiziksel kargolar aynı gün içerisinde anlaşmalı kargoya verilir.</div></details><details class="product-faq-item"><summary class="product-faq-summary">Kurumsal fatura kesebilir misiniz?</summary><div class="product-faq-body">Evet. Sipariş tamamlama adımında şirket unvanı ve vergi numaranızı girdiğiniz takdirde adınıza veya şirketinize e-Arşiv / e-Fatura düzenlenir.</div></details><details class="product-faq-item"><summary class="product-faq-summary">Ödeme adımı güvenli mi?</summary><div class="product-faq-body">HydroGo Commerce modülü 256-bit SSL sertifikası, 3D Secure doğrulama ve PCI-DSS standartlarına uygun ödeme sağlayıcıları (Stripe, iyzico, PayTR) ile çalışır. Kart bilgileriniz sunucularımızda saklanmaz.</div></details></div></div></div><div class="product-side-col"><div class="product-sticky-box"><div class="product-kicker-bar"><span class="product-sku">%s</span><div class="product-rating-badge"><span class="product-rating-stars">★★★★★</span><span>4.9 (48)</span></div></div><h1>%s</h1><div class="product-price-panel"><div class="product-price-main"><span class="product-price-value">%.2f %s</span><span class="product-vat-tag">Tüm vergiler (KDV) dahildir</span></div><div class="product-stock-indicator"><span class="pulse-green-dot"></span><span>%d Adet Stokta</span></div></div><form class="product-order-form" method="post" action="/shop/cart/items" style="display:grid;gap:18px;"><input type="hidden" name="_csrf" value="%s"><input type="hidden" name="product_id" value="%s">%s<div class="qty-btn-group" style="display:flex;align-items:center;justify-content:space-between;background:var(--theme-soft);border:1px solid var(--theme-line);padding:10px 14px;border-radius:var(--theme-radius-md);"><label style="display:flex;align-items:center;gap:10px;font-weight:700;font-size:0.9rem;color:var(--theme-ink);"><span>Sipariş Adedi:</span></label><input name="quantity" type="number" min="1" max="1000" value="1" style="width:75px;padding:6px 10px;border:1px solid #cbd5e1;border-radius:8px;font-weight:700;text-align:center;"></div>%s</form><ul class="product-perks-checklist"><li><span class="perk-check">✓</span> Anında onay ve sipariş takibi</li><li><span class="perk-check">✓</span> 3D Secure güvenli ödeme koruması</li><li><span class="perk-check">✓</span> Kurumsal e-fatura &amp; muhasebe uyumlu</li><li><span class="perk-check">✓</span> 7/24 Kesintisiz teknik danışma</li></ul><div class="product-support-box"><span style="font-size:1.2rem;">💬</span><div><strong>Destek mi lazım?</strong><div style="font-size:0.78rem;color:#3b82f6;">Sorularınız için bizimle iletişime geçin.</div></div></div></div></div></div>%s</main>%s`,
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
	_, _ = fmt.Fprint(w, commerceDocumentStart("Alışveriş Sepetim · HydroGo"))
	_, _ = fmt.Fprintf(w, `<main class="news-portal cart-page"><div class="cart-shell"><div class="news-article-kicker">ALIŞVERİŞ SEPETİNİZ</div><h1 class="news-article-title">Sepetim</h1>`)
	if len(cart.Items) == 0 {
		_, _ = w.Write([]byte(`<div class="news-empty-state" style="text-align:center;padding:48px 20px;"><div class="empty-icon" style="font-size:3.5rem;margin-bottom:12px;">🛒</div><h2 style="font-size:1.4rem;font-weight:700;color:var(--theme-ink);margin:0 0 8px;">Sepetiniz boş</h2><p style="color:var(--theme-muted);font-size:1rem;margin:0 0 24px;">Kataloğumuzdaki zengin ürünleri inceleyip sepetinize ekleyebilirsiniz.</p><a class="btn-primary" href="/shop">Alışverişe Başla →</a></div>`))
	} else {
		_, _ = w.Write([]byte(`<ul class="cart-list">`))
		for _, item := range cart.Items {
			_, _ = fmt.Fprintf(w, `<li class="cart-item"><div class="cart-item-copy"><strong>%s</strong><span>%d adet × %.2f %s</span></div><div class="cart-item-actions"><strong>%.2f %s</strong><form method="post" action="/shop/cart/items/%s/remove"><input type="hidden" name="_csrf" value="%s"><button type="submit" class="button button-quiet button-danger">Kaldır</button></form></div></li>`, template.HTMLEscapeString(item.Title), item.Quantity, float64(item.UnitCents)/100.0, template.HTMLEscapeString(cart.Currency), float64(item.TotalCents)/100.0, template.HTMLEscapeString(cart.Currency), template.HTMLEscapeString(item.ProductID), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())))
		}
		_, _ = fmt.Fprintf(w, `</ul><div class="cart-summary"><div><span>Toplam</span><strong>%.2f %s</strong></div><a class="btn-primary" href="/shop/checkout">Siparişi tamamla →</a></div>`, float64(cart.TotalCents)/100.0, template.HTMLEscapeString(cart.Currency))
	}
	_, _ = fmt.Fprintf(w, `</div></main>%s`, commerceEnterpriseFooter())
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
	_, _ = fmt.Fprint(w, commerceDocumentStart("Ödeme ve Sipariş Onayı · HydroGo"))
	_, _ = fmt.Fprintf(w, `<main class="news-portal" style="max-width:760px;"><div class="news-article-card" style="background:#fff;border:1px solid var(--theme-line);border-radius:var(--theme-radius-xl);padding:clamp(24px,4vw,40px);box-shadow:var(--theme-shadow-xs);"><div class="news-article-kicker" style="color:var(--theme-primary);font-size:.78rem;font-weight:800;letter-spacing:.12em;margin-bottom:8px;">GÜVENLİ ÖDEME ADIMI</div><h1 class="news-article-title" style="font-size:2rem;font-weight:800;letter-spacing:-.03em;color:var(--theme-ink);margin-bottom:8px;">Teslimat ve Sipariş Bilgileri</h1><p style="color:var(--theme-muted);margin-bottom:24px;">Toplam Sipariş Tutarı: <strong style="color:var(--theme-primary-dark);font-size:1.3rem;">%.2f %s</strong></p><form class="studio-form" method="post" action="/shop/checkout"><input type="hidden" name="_csrf" value="%s"><label>Ad Soyad <input name="customer_name" required maxlength="120" placeholder="Müşteri Adı Soyadı"></label><label>T.C. Kimlik / Vergi No <input name="customer_identity_number" required maxlength="20" placeholder="11 haneli kimlik no"></label><div style="display:grid;grid-template-columns:1fr 1fr;gap:14px;"><label>E-Posta <input name="customer_email" type="email" required maxlength="254" placeholder="musteri@example.com"></label><label>Telefon <input name="customer_phone" required maxlength="32" placeholder="+90 555 000 0000"></label></div><label>Teslimat Adresi <textarea name="customer_address" required maxlength="500" rows="3" placeholder="Sokak, Mahalle, Bina ve Daire no"></textarea></label><div style="display:grid;grid-template-columns:2fr 1fr 1fr;gap:14px;"><label>Şehir <input name="customer_city" required maxlength="100" placeholder="İstanbul"></label><label>Ülke Kodu <input name="customer_country" value="TR" required maxlength="2"></label><label>Posta Kodu <input name="customer_postal_code" maxlength="20" placeholder="34000"></label></div><button class="btn-primary" type="submit" style="width:100%%;justify-content:center;padding:14px;font-size:1rem;margin-top:12px;">Siparişi Oluştur ve Ödemeye Geç 💳</button></form></div></main>%s`, float64(cart.TotalCents)/100.0, template.HTMLEscapeString(cart.Currency), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), commerceEnterpriseFooter())
}

func commerceDocumentStart(title string) string {
	return fmt.Sprintf(`<!doctype html><html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>%s</title><link rel="preconnect" href="https://fonts.googleapis.com"><link rel="preconnect" href="https://fonts.gstatic.com" crossorigin><link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&amp;family=JetBrains+Mono:wght@400;500;600;700&amp;display=block"><link rel="stylesheet" href="/theme/static/theme.css?v=20260912"></head><body><div class="site-announcement"><div class="announcement-content"><span class="announcement-badge">Yeni Sürüm v2.0</span><span class="announcement-text">HydroGo Çift Veritabanı ve Modüler Mimari Motoru Yayında!</span><a class="announcement-link" href="/public/news/tr/html">Detayları İncele →</a></div></div><header class="news-header"><div class="header-container"><a class="news-logo" href="/public/pages/tr/demo/html"><div class="logo-mark"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg></div><div class="logo-text"><span>HYDRO<b>GO</b></span><small>COMMERCE</small></div></a><nav class="news-nav"><a href="/public/pages/tr/demo/html">Ana Sayfa</a><a href="/public/pages/tr/showcase/html">Showcase</a><a href="/public/pages/tr/about/html">Hakkında</a><a href="/public/news/tr/html">Haberler</a><a href="/shop" class="is-active">Mağaza</a><a href="/shop/cart">🛒 Sepetim</a><a href="/member/login">Giriş</a><a class="nav-admin-btn" href="/admin">Studio Paneli ↗</a></nav></div></header>`, template.HTMLEscapeString(title))
}

func commerceEnterpriseFooter() string {
	return `<footer class="news-footer"><div class="footer-container"><div class="footer-brand-col"><div class="footer-logo"><div class="logo-mark"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"></polygon><polyline points="2 17 12 22 22 17"></polyline><polyline points="2 12 12 17 22 12"></polyline></svg></div><strong>HYDROGO <span>PLATFORM</span></strong></div><p>Yüksek performanslı Go altyapısı, katı mimari sınırlar ve sözleşme testli çift veritabanı desteği ile kurumsal içerik ve e-ticaret platformu.</p><div class="footer-badge-list"><span class="f-badge">⚡ Go 1.22+</span><span class="f-badge">🛡️ Tip Güvenli</span><span class="f-badge">📦 Modüler Core</span></div></div><div class="footer-nav-col"><h4>Platform</h4><nav><a href="/public/pages/tr/demo/html">Ana Sayfa</a><a href="/public/pages/tr/showcase/html">Showcase</a><a href="/public/pages/tr/about/html">Hakkında</a><a href="/public/news/tr/html">Haberler & Blog</a></nav></div><div class="footer-nav-col"><h4>Çözümler</h4><nav><a href="/shop">E-Ticaret Mağazası</a><a href="/admin">Admin Studio Paneli</a><a href="/api/v1/pages">REST API v1</a></nav></div></div><div class="footer-bottom-bar"><p>© 2026 HydroGo Platform. Tüm hakları saklıdır.</p><div class="footer-sub-links"><a href="/public/pages/tr/about/html">Güvenlik & Felsefe</a><span>•</span><a href="/admin">Yönetim Girişi</a></div></div></footer><script src="/theme/static/theme.js?v=20260912"></script></body></html>`
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
	_, _ = fmt.Fprintf(w, `<!doctype html><html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Sipariş Alındı · HydroGo</title><link rel="preconnect" href="https://fonts.googleapis.com"><link rel="preconnect" href="https://fonts.gstatic.com" crossorigin><link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=block"><link rel="stylesheet" href="/theme/static/theme.css?v=20260910"></head><body><header class="news-header"><div class="header-container"><a class="news-logo" href="/public/pages/tr/demo/html"><div class="logo-mark"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg></div><div class="logo-text"><span>HYDRO<b>GO</b></span><small>SİPARİŞ</small></div></a><nav class="news-nav"><a href="/shop">Mağaza</a><a href="/public/news/tr/html">Haberler</a></nav></div></header><main class="news-portal" style="max-width:760px;"><div class="news-article-card"><div class="news-article-kicker" style="color:#16a34a;">SİPARİŞİNİZ OLUŞTURULDU</div><h1 class="news-article-title" style="margin-bottom:12px;">Siparişiniz Başarıyla Alındı</h1><p style="color:var(--theme-muted);margin-bottom:24px;">Sipariş Kodu: <code style="font-family:'JetBrains Mono',monospace;background:#f1f5f9;padding:3px 8px;border-radius:6px;font-size:.95rem;color:var(--theme-primary-dark);">%s</code> &bull; Durum: <strong>%s</strong></p><div style="background:#f8fafc;border:1px solid #e2e8f0;border-radius:14px;padding:24px;margin-bottom:28px;"><h3 style="margin:0 0 12px;font-size:1.1rem;color:var(--theme-ink);">Ödeme Başlatma</h3><p style="margin:0 0 16px;color:var(--theme-muted);font-size:.92rem;">Test ve simülasyon amacıyla ödeme sağlayıcınızı seçip provizyon alabilirsiniz:</p><form id="payment-form" style="display:grid;gap:14px;"><input type="hidden" id="csrf-token" value="%s"><label style="font-weight:600;color:var(--theme-ink);display:grid;gap:6px;">Ödeme Sağlayıcı <select id="payment-provider"><option value="stripe">Stripe</option><option value="iyzico">iyzico Checkout Form</option><option value="paytr">PayTR Direct Token</option></select></label><button id="payment-submit" class="btn-primary" type="submit" style="justify-content:center;">Ödemeyi Başlat 💳</button></form><p id="payment-status" role="status" style="margin:16px 0 0;font-weight:600;color:var(--theme-primary);"></p><pre id="payment-token" hidden style="margin-top:12px;padding:12px;background:#0f172a;color:#38bdf8;border-radius:8px;font-size:.85rem;overflow-x:auto;"></pre></div><a class="btn-secondary" href="/shop">← Alışverişe Geri Dön</a></div></main><script>(function(){const form=document.getElementById('payment-form');const button=document.getElementById('payment-submit');const status=document.getElementById('payment-status');const token=document.getElementById('payment-token');form.addEventListener('submit',async function(event){event.preventDefault();button.disabled=true;status.textContent='Ödeme başlatılıyor…';token.hidden=true;try{const response=await fetch('/api/commerce/orders/%s/payment-intent',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':document.getElementById('csrf-token').value},body:JSON.stringify({provider:document.getElementById('payment-provider').value})});const data=await response.json().catch(function(){return {};});if(!response.ok)throw new Error(data.error||'Ödeme başlatılamadı');if(data.redirect_url){window.location.assign(data.redirect_url);return;}status.textContent='Ödeme talebi hazırlandı. Seçilen sağlayıcı ile devam ediliyor.';token.textContent=data.client_token||data.id||'';token.hidden=!token.textContent;}catch(error){status.textContent=error.message||'Ödeme başlatılamadı.';button.disabled=false;}})})();</script></body></html>`, template.HTMLEscapeString(order.ID), template.HTMLEscapeString(order.PaymentStatus), template.HTMLEscapeString(security.CSRFTokenFromContext(r.Context())), template.HTMLEscapeString(order.ID))
}

func productMediaMarkup(product commerce.Product) template.HTML {
	if strings.TrimSpace(product.MediaID) == "" {
		return ""
	}
	return template.HTML(`<img class="product-image" src="/media/` + template.URLQueryEscaper(product.MediaID) + `?variant=card" alt="` + template.HTMLEscapeString(product.Title) + `" loading="lazy" onerror="this.onerror=null;this.src='/media/` + template.URLQueryEscaper(product.MediaID) + `';">`)
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
	if h.Limiter != nil && !h.Limiter.Allow("checkout:"+user.ID) {
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
