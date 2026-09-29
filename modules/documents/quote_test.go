package documents

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCalculateQuoteTotalsUsesMinorUnitsAndTaxRounding(t *testing.T) {
	totals, err := CalculateQuoteTotals([]QuoteItem{{Description: "Consulting", Quantity: 3, UnitPriceMinor: 1250, TaxBasisPoints: 1800}})
	if err != nil {
		t.Fatal(err)
	}
	if totals.SubtotalMinor != 3750 || totals.TaxMinor != 675 || totals.TotalMinor != 4425 {
		t.Fatalf("quote totals = %+v", totals)
	}
	if _, err := CalculateQuoteTotals([]QuoteItem{{Description: "unsafe overflow", Quantity: 1_000_000, UnitPriceMinor: 1_000_000_000_000, TaxBasisPoints: 100_000}}); !errors.Is(err, ErrQuoteTooLarge) {
		t.Fatalf("overflow error = %v", err)
	}
}

func TestGenerateQuotePDFIsBoundedAndSupportsUnicode(t *testing.T) {
	quote := Quote{
		Number: "Q-2026-01", Currency: "USD", IssuedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), ExpiresAt: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Company: "HydroGo", CompanyAddress: "1 Main Street", CompanyEmail: "sales@example.test", Customer: "Example Customer", CustomerAddress: "2 Market Street",
		BankDetails: "Account: 123456", Notes: "Payment due within 30 days.", Items: []QuoteItem{{Description: "Implementation services", Quantity: 2, UnitPriceMinor: 25000, TaxBasisPoints: 2000}},
	}
	pdf, err := GenerateQuotePDF(quote)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) || len(pdf) > 8<<20 || !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Fatalf("invalid or unbounded PDF: len=%d header=%q", len(pdf), pdf[:min(5, len(pdf))])
	}
	quote.Company = "Şirket"
	pdf, err = GenerateQuotePDF(quote)
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("Unicode quote PDF failed: len=%d err=%v", len(pdf), err)
	}
	quote.Company = "Company"
	quote.Notes = strings.Repeat("x", 2001)
	if _, err := GenerateQuotePDF(quote); !errors.Is(err, ErrInvalidQuote) {
		t.Fatalf("oversized quote error = %v", err)
	}
}
