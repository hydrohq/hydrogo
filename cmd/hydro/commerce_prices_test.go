package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/modules/commerce"
)

func TestToPriceConvertsTwoDecimalCatalogPricesToCents(t *testing.T) {
	cases := []struct {
		in   any
		want int64
	}{{499.9, 49990}, {0.1, 10}, {19.99, 1999}, {1234.565, 123457}, {float64(10), 1000}}
	for _, tc := range cases {
		price, err := toPrice(tc.in, "TRY")
		if err != nil || price.Cents != tc.want || price.Currency != "TRY" {
			t.Errorf("toPrice(%v) = %+v, %v; want %d TRY", tc.in, price, err, tc.want)
		}
	}
	for _, bad := range []any{nil, "499.90", -1.0} {
		if _, err := toPrice(bad, "TRY"); !errors.Is(err, commerce.ErrUnknownVariant) {
			t.Errorf("toPrice(%v) must refuse to sell, got %v", bad, err)
		}
	}
}

func TestRequireStaffPermissionRefusesAStaffUserWithoutIt(t *testing.T) {
	reached := false
	guarded := requireStaffPermission(auth.PermissionCommerceWrite, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true }))

	viewer := &auth.User{ID: "u-viewer", Role: auth.RoleViewer, Active: true}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/commerce/discount-codes", nil)
	rec := httptest.NewRecorder()
	guarded.ServeHTTP(rec, req.WithContext(auth.WithUser(req.Context(), viewer)))
	if rec.Code != http.StatusForbidden || reached {
		t.Fatalf("viewer creating a discount code: status=%d reached=%v", rec.Code, reached)
	}

	admin := &auth.User{ID: "u-admin", Role: auth.RoleAdmin, Active: true}
	rec = httptest.NewRecorder()
	guarded.ServeHTTP(rec, req.WithContext(auth.WithUser(req.Context(), admin)))
	if !reached {
		t.Fatal("an admin must pass the commerce write check")
	}
}
