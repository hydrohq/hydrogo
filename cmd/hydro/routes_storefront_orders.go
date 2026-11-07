package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/api"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func (a *serveApp) storefrontOrderHandler(w http.ResponseWriter, r *http.Request) {
	type cartItemIn struct {
		ID    string  `json:"id"`
		Title string  `json:"title"`
		Price float64 `json:"price"`
		Qty   int     `json:"qty"`
		Size  string  `json:"size"`
		Color string  `json:"color"`
	}

	var in struct {
		ProductID       string       `json:"product_id"`
		ProductTitle    string       `json:"product_title"`
		Price           float64      `json:"price"`
		Size            string       `json:"size"`
		Color           string       `json:"color"`
		SKU             string       `json:"sku"`
		VariantTitle    string       `json:"variant_title"`
		Items           []cartItemIn `json:"items"`
		Discount        float64      `json:"discount"`
		PromoCode       string       `json:"promo_code"`
		CustomerName    string       `json:"customer_name"`
		CustomerPhone   string       `json:"customer_phone"`
		CustomerAddress string       `json:"customer_address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		api.WriteError(w, r, http.StatusBadRequest, "invalid json payload", nil)
		return
	}

	// 1. Resolve tenant scope
	scope, err := tenant.FromContext(r.Context())
	tenantFound := err == nil && scope.TenantID != ""
	if !tenantFound {
		host := strings.ToLower(strings.TrimSpace(r.Host))
		if i := strings.IndexByte(host, ':'); i > 0 {
			host = host[:i]
		}
		if res, resOk := tenant.ResolveTenantByDomain(r.Context(), a.db, string(a.cfg.Database.Dialect), host); resOk {
			scope = res
			tenantFound = true
		}
	}

	var codeBytes [2]byte
	_, _ = rand.Read(codeBytes[:])
	codeNum := 1000 + (int(codeBytes[0])<<8|int(codeBytes[1]))%9000
	orderCode := fmt.Sprintf("#HYD-%04d", codeNum)

	var totalPrice float64 = in.Price
	var itemsSummary string

	if len(in.Items) > 0 {
		var sb strings.Builder
		totalPrice = 0
		for i, item := range in.Items {
			qty := item.Qty
			if qty <= 0 {
				qty = 1
			}
			totalPrice += item.Price * float64(qty)
			if i > 0 {
				sb.WriteString(", ")
			}
			detail := item.Title
			if item.Size != "" && item.Size != "Standart" {
				detail += " (" + item.Size + ")"
			}
			if item.Color != "" {
				detail += " (" + item.Color + ")"
			}
			sb.WriteString(fmt.Sprintf("%dx %s", qty, detail))
		}
		if in.Discount > 0 {
			totalPrice -= in.Discount
			if totalPrice < 0 {
				totalPrice = 0
			}
			sb.WriteString(fmt.Sprintf(" [Kupon: %s -%.2f TL]", in.PromoCode, in.Discount))
		}
		itemsSummary = sb.String()
	} else {
		variantSummary := in.Size
		if in.Color != "" {
			if variantSummary != "" {
				variantSummary = in.Color + " / " + variantSummary
			} else {
				variantSummary = in.Color
			}
		}
		if in.VariantTitle != "" {
			variantSummary = in.VariantTitle
		}
		if variantSummary != "" {
			itemsSummary = fmt.Sprintf("%s (%s)", in.ProductTitle, variantSummary)
		} else {
			itemsSummary = in.ProductTitle
		}
	}

	if tenantFound && scope.TenantID != "" {
		tenantCtx := tenant.WithScope(r.Context(), scope)

		orderData := map[string]any{
			"order_code":       orderCode,
			"order_no":         orderCode,
			"customer_name":    in.CustomerName,
			"customer_phone":   in.CustomerPhone,
			"customer_address": in.CustomerAddress,
			"shipping_address": in.CustomerAddress,
			"items_json":       itemsSummary,
			"items_summary":    itemsSummary,
			"total_price":      totalPrice,
			"total_amount":     totalPrice,
			"status":           "yeni",
			"order_channel":    "WhatsApp",
		}

		// Try orders collection, then boutique_orders
		created, err := a.collectionService.CreateRecord(tenantCtx, "orders", orderData)
		if err != nil {
			created, err = a.collectionService.CreateRecord(tenantCtx, "boutique_orders", orderData)
		}
		if err == nil && created.Data != nil {
			if codeVal, ok := created.Data["order_code"].(string); ok && codeVal != "" {
				orderCode = codeVal
			}
		}
	}

	api.WriteData(w, http.StatusOK, map[string]any{
		"order_code": orderCode,
		"status":     "PENDING_WHATSAPP",
	}, nil)
}
