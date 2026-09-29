package documents

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/signintech/gopdf"
	"golang.org/x/image/font/gofont/goregular"
)

var currencyCode = regexp.MustCompile(`^[A-Z]{3}$`)

var (
	ErrInvalidQuote  = errors.New("invalid quote")
	ErrQuoteTooLarge = errors.New("quote exceeds document limits")
)

type Quote struct {
	Number, Currency                                   string
	IssuedAt, ExpiresAt                                time.Time
	Company, CompanyAddress, CompanyEmail, BankDetails string
	Customer, CustomerAddress                          string
	Notes                                              string
	Items                                              []QuoteItem
	LogoPNGOrJPEG                                      []byte
	FontTTF                                            []byte
}

type QuoteItem struct {
	Description    string
	Quantity       int64
	UnitPriceMinor int64
	TaxBasisPoints int64
}

type QuoteTotals struct{ SubtotalMinor, TaxMinor, TotalMinor int64 }

func CalculateQuoteTotals(items []QuoteItem) (QuoteTotals, error) {
	if len(items) == 0 || len(items) > 100 {
		return QuoteTotals{}, ErrInvalidQuote
	}
	var totals QuoteTotals
	for _, item := range items {
		if strings.TrimSpace(item.Description) == "" || len(item.Description) > 180 || item.Quantity < 1 || item.Quantity > 1_000_000 || item.UnitPriceMinor < 0 || item.UnitPriceMinor > 1_000_000_000_000 || item.TaxBasisPoints < 0 || item.TaxBasisPoints > 100_000 {
			return QuoteTotals{}, ErrInvalidQuote
		}
		if item.UnitPriceMinor != 0 && item.Quantity > math.MaxInt64/item.UnitPriceMinor {
			return QuoteTotals{}, ErrQuoteTooLarge
		}
		line := item.Quantity * item.UnitPriceMinor
		if totals.SubtotalMinor > math.MaxInt64-line {
			return QuoteTotals{}, ErrQuoteTooLarge
		}
		if item.TaxBasisPoints != 0 && line > (math.MaxInt64-5000)/item.TaxBasisPoints {
			return QuoteTotals{}, ErrQuoteTooLarge
		}
		tax := (line*item.TaxBasisPoints + 5000) / 10000
		if totals.TaxMinor > math.MaxInt64-tax {
			return QuoteTotals{}, ErrQuoteTooLarge
		}
		totals.SubtotalMinor += line
		totals.TaxMinor += tax
	}
	if totals.SubtotalMinor > math.MaxInt64-totals.TaxMinor {
		return QuoteTotals{}, ErrQuoteTooLarge
	}
	totals.TotalMinor = totals.SubtotalMinor + totals.TaxMinor
	return totals, nil
}

// GenerateQuotePDF renders a bounded, self-contained quotation without an
// external renderer. A caller-supplied TrueType font enables full Unicode text.
func GenerateQuotePDF(quote Quote) ([]byte, error) {
	if err := validateQuote(quote); err != nil {
		return nil, err
	}
	totals, err := CalculateQuoteTotals(quote.Items)
	if err != nil {
		return nil, err
	}
	var pdf gopdf.GoPdf
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4, Unit: gopdf.UnitMM})
	font := quote.FontTTF
	if len(font) == 0 {
		font = goregular.TTF
	}
	if err := pdf.AddTTFFontByReader("HydroGo", bytes.NewReader(font)); err != nil {
		return nil, ErrInvalidQuote
	}
	if err := pdf.SetFont("HydroGo", "", 10); err != nil {
		return nil, ErrInvalidQuote
	}
	pdf.AddPage()
	pdf.SetFillColor(25, 65, 61)
	pdf.RectFromUpperLeftWithStyle(0, 0, 210, 42, "F")
	pdf.SetTextColor(255, 255, 255)
	if len(quote.LogoPNGOrJPEG) > 0 {
		config, _, err := image.DecodeConfig(bytes.NewReader(quote.LogoPNGOrJPEG))
		if err != nil || config.Width < 1 || config.Height < 1 || int64(config.Width)*int64(config.Height) > 16_000_000 {
			return nil, ErrInvalidQuote
		}
		logo, _, err := image.Decode(bytes.NewReader(quote.LogoPNGOrJPEG))
		if err != nil {
			return nil, ErrInvalidQuote
		}
		if err := pdf.ImageFrom(logo, 15, 10, &gopdf.Rect{W: 24, H: 20}); err != nil {
			return nil, ErrInvalidQuote
		}
	}
	pdf.SetXY(45, 13)
	if err := pdf.Cell(&gopdf.Rect{W: 145, H: 8}, quote.Company); err != nil {
		return nil, ErrInvalidQuote
	}
	pdf.SetFont("HydroGo", "", 18)
	pdf.SetTextColor(35, 45, 50)
	pdf.SetXY(15, 51)
	_ = pdf.Cell(&gopdf.Rect{W: 180, H: 12}, "QUOTE / PROFORMA")
	pdf.SetFont("HydroGo", "", 10)
	writeAt := func(x, y float64, text string) error {
		pdf.SetTextColor(35, 45, 50)
		pdf.SetXY(x, y)
		return pdf.Cell(&gopdf.Rect{W: 180 - x + 15, H: 7}, text)
	}
	if err := writeAt(15, 67, "Quote: "+quote.Number); err != nil {
		return nil, ErrInvalidQuote
	}
	if err := writeAt(110, 67, "Currency: "+quote.Currency); err != nil {
		return nil, ErrInvalidQuote
	}
	if err := writeAt(15, 76, "Issued: "+quote.IssuedAt.Format("2006-01-02")); err != nil {
		return nil, ErrInvalidQuote
	}
	if err := writeAt(110, 76, "Valid through: "+quote.ExpiresAt.Format("2006-01-02")); err != nil {
		return nil, ErrInvalidQuote
	}
	if err := writeAt(15, 91, "From: "+quote.CompanyAddress); err != nil {
		return nil, ErrInvalidQuote
	}
	if err := writeAt(15, 99, "Email: "+quote.CompanyEmail); err != nil {
		return nil, ErrInvalidQuote
	}
	if err := writeAt(15, 114, "Prepared for: "+quote.Customer); err != nil {
		return nil, ErrInvalidQuote
	}
	if err := writeAt(15, 122, "Address: "+quote.CustomerAddress); err != nil {
		return nil, ErrInvalidQuote
	}
	columns := []struct {
		x, w  float64
		label string
	}{{15, 78, "Description"}, {93, 18, "Qty"}, {111, 32, "Unit price"}, {143, 20, "Tax"}, {163, 32, "Total"}}
	pdf.SetFillColor(232, 240, 239)
	pdf.RectFromUpperLeftWithStyle(15, 137, 180, 10, "F")
	for _, column := range columns {
		pdf.SetXY(column.x, 139)
		if err := pdf.Cell(&gopdf.Rect{W: column.w, H: 7}, column.label); err != nil {
			return nil, ErrInvalidQuote
		}
	}
	y := 149.0
	for _, item := range quote.Items {
		if y > 250 {
			pdf.AddPage()
			y = 25
		}
		line := item.Quantity * item.UnitPriceMinor
		values := []string{item.Description, fmt.Sprint(item.Quantity), money(item.UnitPriceMinor, quote.Currency), fmt.Sprintf("%.2f%%", float64(item.TaxBasisPoints)/100), money(line, quote.Currency)}
		for i, column := range columns {
			pdf.SetXY(column.x, y)
			if err := pdf.Cell(&gopdf.Rect{W: column.w, H: 7}, values[i]); err != nil {
				return nil, ErrInvalidQuote
			}
		}
		y += 9
	}
	if y > 252 {
		pdf.AddPage()
		y = 25
	}
	for i, row := range []struct {
		label string
		value int64
	}{{"Subtotal", totals.SubtotalMinor}, {"Tax", totals.TaxMinor}, {"Total", totals.TotalMinor}} {
		pdf.SetXY(125, y+float64(i)*8)
		if err := pdf.Cell(&gopdf.Rect{W: 35, H: 7}, row.label); err != nil {
			return nil, ErrInvalidQuote
		}
		pdf.SetXY(160, y+float64(i)*8)
		if err := pdf.Cell(&gopdf.Rect{W: 35, H: 7}, money(row.value, quote.Currency)); err != nil {
			return nil, ErrInvalidQuote
		}
	}
	y += 32
	for _, section := range []struct{ label, value string }{{"Bank details", quote.BankDetails}, {"Notes", quote.Notes}} {
		if section.value == "" {
			continue
		}
		if y > 270 {
			pdf.AddPage()
			y = 25
		}
		pdf.SetXY(15, y)
		_ = pdf.Cell(&gopdf.Rect{W: 180, H: 7}, section.label)
		y += 7
		pdf.SetXY(15, y)
		_ = pdf.MultiCellWithOption(&gopdf.Rect{W: 180, H: 25}, section.value, gopdf.CellOption{})
		y += 18
	}
	if quote.CompanyEmail != "" {
		pdf.SetXY(15, 286)
		_ = pdf.Cell(&gopdf.Rect{W: 180, H: 6}, quote.CompanyEmail)
	}
	data, err := pdf.GetBytesPdfReturnErr()
	if err != nil || len(data) > 8<<20 {
		return nil, ErrQuoteTooLarge
	}
	return data, nil
}

func validateQuote(q Quote) error {
	if len(q.Number) == 0 || len(q.Number) > 64 || strings.ContainsAny(q.Number, "\r\n") || !currencyCode.MatchString(q.Currency) || q.IssuedAt.IsZero() || q.ExpiresAt.Before(q.IssuedAt) || len(q.Company) == 0 || len(q.Company) > 160 || len(q.CompanyAddress) > 500 || len(q.CompanyEmail) > 254 || len(q.BankDetails) > 1000 || len(q.Customer) == 0 || len(q.Customer) > 160 || len(q.CustomerAddress) > 500 || len(q.Notes) > 2000 || len(q.Items) == 0 || len(q.Items) > 100 || len(q.LogoPNGOrJPEG) > 2<<20 || len(q.FontTTF) > 8<<20 {
		return ErrInvalidQuote
	}
	return nil
}
func money(minor int64, currency string) string {
	return fmt.Sprintf("%s %d.%02d", currency, minor/100, minor%100)
}
