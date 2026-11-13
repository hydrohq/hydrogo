package commerce

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// DefaultIndividualTaxNo is the standard Turkish GİB tax identifier used when an
// individual shopper leaves TCKN blank at checkout.
const DefaultIndividualTaxNo = "11111111111"

// InvoiceProvider issues sales/e-Arşiv/e-Fatura documents in an external
// accounting system without commerce importing any vendor SDK or HTTP client.
type InvoiceProvider interface {
	Name() string
	IssueInvoice(ctx context.Context, doc InvoiceDocument) (InvoiceResult, error)
}

// InvoiceCustomerInfo holds customer tax registration details collected at
// checkout (individual TCKN or corporate VKN/tax office).
type InvoiceCustomerInfo struct {
	InvoiceType string
	TCKN        string
	CompanyName string
	VKN         string
	TaxOffice   string
}

// InvoiceCustomerResolver loads optional tax details for an order (for example
// from the invoice_details collection).
type InvoiceCustomerResolver interface {
	ResolveInvoiceCustomer(ctx context.Context, orderID string) (InvoiceCustomerInfo, bool, error)
}

// InvoiceResult is returned by InvoiceProvider once an invoice document is created.
type InvoiceResult struct {
	Provider  string `json:"provider,omitempty"`
	GUID      string `json:"guid"`
	InvoiceNo string `json:"invoice_no,omitempty"`
	URL       string `json:"url,omitempty"`
}

// InvoiceCustomer holds resolved buyer identity and tax fields on InvoiceDocument.
type InvoiceCustomer struct {
	CustomerID string
	Title      string
	Address    string
	TaxOffice  string
	TaxNo      string
	Email      string
	Phone      string
	IsCompany  bool
}

// InvoiceLine holds one pre-tax, tax, and tax-inclusive line item in integer cents
// so accounting adapters never lose a kuruş to floating-point accumulation.
type InvoiceLine struct {
	ProductID        string
	SKU              string
	ProductName      string
	Note             string
	TaxRatePct       int
	Quantity         int
	UnitPriceCents   int64
	GrossCents       int64
	DiscountNetCents int64
	NetCents         int64
	TaxCents         int64
	TotalCents       int64
}

// InvoiceAmounts aggregates all lines so Σ Details == Amounts always holds.
type InvoiceAmounts struct {
	Currency      string
	GrossCents    int64
	DiscountCents int64
	NetCents      int64
	TaxCents      int64
	TotalCents    int64
}

// InvoiceDocument is the provider-neutral sales invoice payload for an order.
type InvoiceDocument struct {
	OrderID      string
	InvoiceNo    string
	IssuedAt     time.Time
	Currency     string
	Note         string
	Customer     InvoiceCustomer
	Lines        []InvoiceLine
	Amounts      InvoiceAmounts
	TaxInclusive bool
}

// BuildInvoiceDocument constructs a kuruş-balanced invoice document from an order,
// its items, configured pricing/tax rules, and optional customer tax info.
func BuildInvoiceDocument(order Order, items []OrderItem, pricing PricingConfig, taxInfo InvoiceCustomerInfo, hasTaxInfo bool, issuedAt time.Time) (InvoiceDocument, error) {
	if order.TotalCents <= 0 {
		return InvoiceDocument{}, fmt.Errorf("%w: order total must be positive", ErrOrderNotInvoiceable)
	}
	if len(items) == 0 {
		return InvoiceDocument{}, fmt.Errorf("%w: order has no line items", ErrOrderNotInvoiceable)
	}
	if issuedAt.IsZero() {
		issuedAt = time.Now().UTC()
	}

	customer, err := resolveCustomer(order, taxInfo, hasTaxInfo)
	if err != nil {
		return InvoiceDocument{}, err
	}

	taxRatePct := int(math.Round(pricing.TaxRate * 100))
	if taxRatePct < 0 {
		taxRatePct = 0
	}

	var lines []InvoiceLine
	if pricing.TaxInclusive || taxRatePct == 0 {
		lines = buildInclusiveLines(order, items, taxRatePct)
	} else {
		lines = buildExclusiveLines(order, items, taxRatePct)
	}

	var amounts InvoiceAmounts
	amounts.Currency = normalizeInvoiceCurrency(order.Currency)
	for _, line := range lines {
		amounts.GrossCents += line.GrossCents
		amounts.DiscountCents += line.DiscountNetCents
		amounts.NetCents += line.NetCents
		amounts.TaxCents += line.TaxCents
		amounts.TotalCents += line.TotalCents
	}

	note := fmt.Sprintf("Sipariş #%s", order.ID)
	if strings.TrimSpace(order.DiscountCode) != "" {
		note += fmt.Sprintf(" (%s)", strings.TrimSpace(order.DiscountCode))
	}

	return InvoiceDocument{
		OrderID:      order.ID,
		InvoiceNo:    order.ID,
		IssuedAt:     issuedAt,
		Currency:     amounts.Currency,
		Note:         note,
		Customer:     customer,
		Lines:        lines,
		Amounts:      amounts,
		TaxInclusive: pricing.TaxInclusive,
	}, nil
}

func resolveCustomer(order Order, taxInfo InvoiceCustomerInfo, hasTaxInfo bool) (InvoiceCustomer, error) {
	customerID := strings.TrimSpace(order.MemberID)
	if customerID == "" {
		customerID = strings.TrimSpace(order.GuestEmail)
	}
	if customerID == "" {
		customerID = order.ID
	}

	base := InvoiceCustomer{
		CustomerID: customerID,
		Address:    formatInvoiceAddress(order),
		Email:      strings.TrimSpace(order.GuestEmail),
		Phone:      strings.TrimSpace(order.GuestPhone),
	}

	if hasTaxInfo && strings.EqualFold(strings.TrimSpace(taxInfo.InvoiceType), "kurumsal") {
		company := strings.TrimSpace(taxInfo.CompanyName)
		office := strings.TrimSpace(taxInfo.TaxOffice)
		vkn := stripSpaces(taxInfo.VKN)
		if company == "" || office == "" || !isDigits(vkn, 10) {
			return InvoiceCustomer{}, fmt.Errorf("%w: corporate invoice requires company_name, tax_office and 10-digit vkn", ErrInvoiceCustomerInvalid)
		}
		base.IsCompany = true
		base.Title = company
		base.TaxOffice = office
		base.TaxNo = vkn
		return base, nil
	}

	title := strings.TrimSpace(order.GuestName)
	if title == "" {
		title = strings.TrimSpace(order.GuestEmail)
	}
	if title == "" {
		title = "Müşteri"
	}
	tckn := stripSpaces(taxInfo.TCKN)
	if tckn == "" {
		tckn = DefaultIndividualTaxNo
	} else if !isDigits(tckn, 11) {
		return InvoiceCustomer{}, fmt.Errorf("%w: tckn must be 11 digits when provided", ErrInvoiceCustomerInvalid)
	}
	base.Title = title
	base.TaxNo = tckn
	return base, nil
}

func buildInclusiveLines(order Order, items []OrderItem, taxRatePct int) []InvoiceLine {
	rate := float64(taxRatePct) / 100.0
	productTarget := order.TotalCents - order.ShippingCents
	if productTarget < 0 {
		productTarget = 0
	}

	rawTotals := make([]int64, len(items))
	for i, it := range items {
		rawTotals[i] = maxInt64(0, it.TotalCents)
	}
	lineTotals := allocateToTarget(rawTotals, productTarget)

	lines := make([]InvoiceLine, 0, len(items)+1)
	for i, it := range items {
		qty := it.Quantity
		if qty <= 0 {
			qty = 1
		}
		title := lineTitle(it)
		totalCents := lineTotals[i]
		netCents := totalCents
		if taxRatePct > 0 {
			netCents = int64(math.Round(float64(totalCents) / (1.0 + rate)))
		}
		taxCents := totalCents - netCents

		unitNetCents := maxInt64(0, it.UnitPriceCents)
		if taxRatePct > 0 {
			unitNetCents = int64(math.Round(float64(unitNetCents) / (1.0 + rate)))
		}

		var lineQty int
		var lineName string
		var unitPriceCents, grossCents, discountNetCents int64

		hasDiscount := totalCents < rawTotals[i]
		switch {
		case !hasDiscount && unitNetCents*int64(qty) == netCents:
			lineQty = qty
			lineName = title
			unitPriceCents = unitNetCents
			grossCents = netCents
			discountNetCents = 0
		case hasDiscount && unitNetCents*int64(qty) >= netCents:
			lineQty = qty
			lineName = title
			unitPriceCents = unitNetCents
			grossCents = unitNetCents * int64(qty)
			discountNetCents = grossCents - netCents
		case hasDiscount:
			preNet := rawTotals[i]
			if taxRatePct > 0 {
				preNet = int64(math.Round(float64(preNet) / (1.0 + rate)))
			}
			if preNet < netCents {
				preNet = netCents
			}
			lineQty = 1
			lineName = title
			if qty > 1 {
				lineName = fmt.Sprintf("%s (x%d)", title, qty)
			}
			unitPriceCents = preNet
			grossCents = preNet
			discountNetCents = preNet - netCents
		default:
			// Multi-unit tax-inclusive item whose per-unit net division leaves a
			// sub-kuruş rounding delta: keep Qty=1 so UnitPrice*Qty==Gross==Net.
			lineQty = 1
			lineName = title
			if qty > 1 {
				lineName = fmt.Sprintf("%s (x%d)", title, qty)
			}
			unitPriceCents = netCents
			grossCents = netCents
			discountNetCents = 0
		}

		lines = append(lines, InvoiceLine{
			ProductID:        lineProductID(it),
			SKU:              strings.TrimSpace(it.SKU),
			ProductName:      lineName,
			TaxRatePct:       taxRatePct,
			Quantity:         lineQty,
			UnitPriceCents:   unitPriceCents,
			GrossCents:       grossCents,
			DiscountNetCents: discountNetCents,
			NetCents:         netCents,
			TaxCents:         taxCents,
			TotalCents:       totalCents,
		})
	}

	if order.ShippingCents > 0 {
		netCents := order.ShippingCents
		if taxRatePct > 0 {
			netCents = int64(math.Round(float64(order.ShippingCents) / (1.0 + rate)))
		}
		taxCents := order.ShippingCents - netCents
		lines = append(lines, InvoiceLine{
			ProductID:        "SHIPPING",
			SKU:              "SHIPPING",
			ProductName:      "Kargo Ücreti",
			TaxRatePct:       taxRatePct,
			Quantity:         1,
			UnitPriceCents:   netCents,
			GrossCents:       netCents,
			DiscountNetCents: 0,
			NetCents:         netCents,
			TaxCents:         taxCents,
			TotalCents:       order.ShippingCents,
		})
	}
	return lines
}

func buildExclusiveLines(order Order, items []OrderItem, taxRatePct int) []InvoiceLine {
	productNetTarget := order.TotalCents - order.ShippingCents - order.TaxCents
	if productNetTarget < 0 {
		productNetTarget = maxInt64(0, order.SubtotalCents-order.DiscountCents)
	}

	rawTotals := make([]int64, len(items))
	for i, it := range items {
		rawTotals[i] = maxInt64(0, it.TotalCents)
	}
	lineNets := allocateToTarget(rawTotals, productNetTarget)
	lineTaxes := allocateToTarget(lineNets, maxInt64(0, order.TaxCents))

	lines := make([]InvoiceLine, 0, len(items)+1)
	for i, it := range items {
		qty := it.Quantity
		if qty <= 0 {
			qty = 1
		}
		title := lineTitle(it)
		netCents := lineNets[i]
		taxCents := lineTaxes[i]
		unitCents := maxInt64(0, it.UnitPriceCents)

		lineQty := qty
		lineName := title
		var grossCents, discountNetCents int64
		if unitCents*int64(qty) >= netCents {
			grossCents = unitCents * int64(qty)
			discountNetCents = grossCents - netCents
		} else {
			lineQty = 1
			if qty > 1 {
				lineName = fmt.Sprintf("%s (x%d)", title, qty)
			}
			unitCents = netCents
			grossCents = netCents
			discountNetCents = 0
		}

		lines = append(lines, InvoiceLine{
			ProductID:        lineProductID(it),
			SKU:              strings.TrimSpace(it.SKU),
			ProductName:      lineName,
			TaxRatePct:       taxRatePct,
			Quantity:         lineQty,
			UnitPriceCents:   unitCents,
			GrossCents:       grossCents,
			DiscountNetCents: discountNetCents,
			NetCents:         netCents,
			TaxCents:         taxCents,
			TotalCents:       netCents + taxCents,
		})
	}

	if order.ShippingCents > 0 {
		lines = append(lines, InvoiceLine{
			ProductID:        "SHIPPING",
			SKU:              "SHIPPING",
			ProductName:      "Kargo Ücreti",
			TaxRatePct:       0,
			Quantity:         1,
			UnitPriceCents:   order.ShippingCents,
			GrossCents:       order.ShippingCents,
			DiscountNetCents: 0,
			NetCents:         order.ShippingCents,
			TaxCents:         0,
			TotalCents:       order.ShippingCents,
		})
	}
	return lines
}

// allocateToTarget scales non-negative weights proportionally so their sum equals
// target in integer cents, assigning any rounding delta to the largest weight.
func allocateToTarget(weights []int64, target int64) []int64 {
	out := make([]int64, len(weights))
	if len(weights) == 0 || target <= 0 {
		return out
	}
	var sumWeights int64
	largestIdx := 0
	for i, w := range weights {
		if w > weights[largestIdx] {
			largestIdx = i
		}
		if w > 0 {
			sumWeights += w
		}
	}
	if sumWeights <= 0 {
		out[0] = target
		return out
	}
	var allocated int64
	for i, w := range weights {
		if w <= 0 {
			continue
		}
		share := int64(math.Round(float64(target) * float64(w) / float64(sumWeights)))
		if share < 0 {
			share = 0
		}
		out[i] = share
		allocated += share
	}
	delta := target - allocated
	out[largestIdx] += delta
	if out[largestIdx] < 0 {
		// Borrow deficit from other positive entries if rounding over-allocated.
		deficit := -out[largestIdx]
		out[largestIdx] = 0
		for i := range out {
			if deficit == 0 {
				break
			}
			take := minInt64(out[i], deficit)
			out[i] -= take
			deficit -= take
		}
	}
	return out
}

func lineProductID(it OrderItem) string {
	if s := strings.TrimSpace(it.SKU); s != "" {
		return s
	}
	if v := strings.TrimSpace(it.VariantID); v != "" {
		return v
	}
	return strings.TrimSpace(it.ID)
}

func lineTitle(it OrderItem) string {
	if t := strings.TrimSpace(it.Title); t != "" {
		return t
	}
	if s := strings.TrimSpace(it.SKU); s != "" {
		return s
	}
	return "Ürün"
}

func formatInvoiceAddress(order Order) string {
	cityLine := strings.TrimSpace(strings.Join(nonEmpty(order.GuestDistrict, order.GuestCity), " / "))
	tail := strings.TrimSpace(strings.Join(nonEmpty(order.GuestPostalCode, order.GuestCountry), " "))
	return strings.Join(nonEmpty(order.GuestAddress, cityLine, tail), ", ")
}

func normalizeInvoiceCurrency(currency string) string {
	c := strings.ToUpper(strings.TrimSpace(currency))
	if c == "" || c == "TRY" {
		return "TL"
	}
	return c
}

func stripSpaces(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), " ", "")
}

func isDigits(s string, exactLen int) bool {
	if len(s) != exactLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func nonEmpty(parts ...string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
