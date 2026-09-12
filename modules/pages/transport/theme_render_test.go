package transport

import (
	"bytes"
	"encoding/json"
	htmltemplate "html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	themev1 "github.com/ardaozyildirim/hydrogo/extension/v1"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

func TestThemeRenderCorporateShowcaseAndBlocks(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(filename), "..", "..", "..")
	bundle, err := themev1.LoadTheme(filepath.Join(repoRoot, "themes"), "corporate")
	if err != nil {
		t.Fatalf("load corporate theme: %v", err)
	}

	showcaseBlocks := []pages.Block{
		{Type: "hero", Payload: json.RawMessage(`{"title":"HydroGo Blok & Bileşen Vitrini","subtitle":"Hero, CTA, Fiyatlandırma Tablosu, Testimonials ve SSS bileşenleri","button_text":"İncele","button_url":"/shop"}`)},
		{Type: "pricing_table", Payload: json.RawMessage(`{"items":[{"name":"Starter Edition","price_cents":0,"currency":"TRY","features":["SQLite","REST API"]},{"name":"Growth Edition","price_cents":49900,"currency":"TRY","features":["Tüm CMS","Commerce"]}]}`)},
		{Type: "testimonials", Payload: json.RawMessage(`{"items":[{"name":"Mühendis","quote":"Performanslı mimari."}]}`)},
		{Type: "faq", Payload: json.RawMessage(`{"items":[{"question":"Desteklenen bloklar?","answer":"Hero, CTA, vb."}]}`)},
		{Type: "cta", Payload: json.RawMessage(`{"title":"Keşfedin","button_text":"Başla","button_url":"/admin"}`)},
	}

	for _, block := range showcaseBlocks {
		var buf bytes.Buffer
		if err := bundle.RenderBlock(&buf, block); err != nil {
			t.Fatalf("render block %s: %v", block.Type, err)
		}
		if buf.Len() == 0 {
			t.Fatalf("block %s rendered empty", block.Type)
		}
	}

	page := pages.Page{
		ID:     "demo-showcase",
		Locale: "tr",
		Slug:   "showcase",
		Title:  "HydroGo Showcase",
		Blocks: showcaseBlocks,
	}

	var content bytes.Buffer
	for _, block := range page.Blocks {
		var themed bytes.Buffer
		if err := bundle.RenderBlock(&themed, block); err != nil {
			t.Fatalf("render themed block: %v", err)
		}
		content.Write(themed.Bytes())
	}

	rec := httptest.NewRecorder()
	if err := bundle.RenderPage(rec, themev1.PageTemplateData{
		Title:   page.Title,
		Slug:    page.Slug,
		Locale:  page.Locale,
		Content: htmltemplate.HTML(content.String()),
		Slots:   map[string]htmltemplate.HTML{"content": htmltemplate.HTML(content.String())},
	}); err != nil {
		t.Fatalf("render page: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !bytes.Contains([]byte(body), []byte("HydroGo Showcase")) {
		t.Fatalf("page output missing title: %s", body)
	}
	if !bytes.Contains([]byte(body), []byte("pricing-card")) {
		t.Fatalf("page output missing pricing-card: %s", body)
	}
}
