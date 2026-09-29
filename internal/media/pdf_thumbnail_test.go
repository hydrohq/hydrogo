package media

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	"github.com/ardaozyildirim/hydrogo/internal/database"
)

func TestRenderPDFCoverCreatesBoundedPNG(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "source.pdf")
	if err := os.WriteFile(input, minimalPDF(), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := renderPDFCover(t.Context(), input, "cover-id")
	if err != nil {
		t.Fatalf("render cover: %v", err)
	}
	if result.Key != "cover-id.png" || result.Mime != "image/png" || result.Size <= 0 {
		t.Fatalf("result = %#v", result)
	}
	f, err := os.Open(filepath.Join(dir, result.Key))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	config, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatalf("decode cover: %v", err)
	}
	if config.Width != 320 || config.Height != 320 {
		t.Fatalf("cover dimensions = %dx%d, want 320x320", config.Width, config.Height)
	}
}

func TestOptimizerStoresPDFCoverAsMediaVariant(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source.pdf"), minimalPDF(), 0600); err != nil {
		t.Fatal(err)
	}
	repository := &recordingDerivativeRepository{}
	optimizer := &Optimizer{storageDir: dir, variants: repository}
	optimizer.process(optimizeJob{asset: Asset{ID: "pdf-asset", StorageKey: "source.pdf", MimeType: "application/pdf"}})
	if len(repository.created) != 1 || repository.created[0].Kind != VariantPDFCover || repository.created[0].MimeType != "image/png" {
		t.Fatalf("created variants = %+v", repository.created)
	}
	if len(repository.results) != 1 || repository.results[0].status != "ready" || repository.results[0].detail == "" || repository.results[0].size <= 0 {
		t.Fatalf("variant results = %+v", repository.results)
	}
}

func TestUploaderAcceptsOnlyValidatedPDFDocuments(t *testing.T) {
	dir := t.TempDir()
	uploader, err := NewUploader(dir, 2<<20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uploader.SaveDocumentContext(t.Context(), bytes.NewReader(minimalPDF()), "cover.pdf", 1<<20); err != nil {
		t.Fatalf("valid PDF upload: %v", err)
	}
	if _, err := uploader.SaveDocumentContext(t.Context(), bytes.NewReader([]byte("not a pdf")), "cover.pdf", 1<<20); err == nil {
		t.Fatal("invalid PDF accepted")
	}
}

func TestRenderPDFCoverRejectsInvalidAndOversizedInput(t *testing.T) {
	dir := t.TempDir()
	invalid := filepath.Join(dir, "invalid.pdf")
	if err := os.WriteFile(invalid, []byte("%PDF-not-a-document"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := renderPDFCover(t.Context(), invalid, "invalid"); err == nil {
		t.Fatal("malformed PDF was rendered")
	}
	oversized := filepath.Join(dir, "oversized.pdf")
	f, err := os.Create(oversized)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate((10 << 20) + 1); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := renderPDFCover(t.Context(), oversized, "oversized"); err == nil {
		t.Fatal("oversized PDF was rendered")
	}
}

type readyPDFVariantRepo struct {
	httpAssetRepo
	variant Variant
}

func (readyPDFVariantRepo) CreateProcessing(context.Context, database.Executor, Variant) error {
	return nil
}
func (readyPDFVariantRepo) MarkResult(context.Context, database.Executor, string, string, int64, string) error {
	return nil
}
func (r readyPDFVariantRepo) GetReady(_ context.Context, _ database.Executor, _, kind string) (Variant, error) {
	if kind != VariantPDFCover {
		return Variant{}, os.ErrNotExist
	}
	return r.variant, nil
}

func TestPrivatePDFCoverVariantIsInlineButSourceRemainsAttachment(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "source.pdf")
	if err := os.WriteFile(sourcePath, minimalPDF(), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := renderPDFCover(t.Context(), sourcePath, "cover"); err != nil {
		t.Fatal(err)
	}
	uploader, err := NewUploader(dir, 2<<20)
	if err != nil {
		t.Fatal(err)
	}
	assets := readyPDFVariantRepo{
		httpAssetRepo: httpAssetRepo{asset: Asset{ID: "private-pdf", StorageKey: "source.pdf", OriginalFilename: "resume.pdf", MimeType: "application/pdf", Status: "ready", Visibility: VisibilityPrivate}},
		variant:       Variant{StorageKey: "cover.png", MimeType: "image/png", Status: "ready"},
	}
	handler := HTTPHandler{Service: Service{Uploader: uploader, Assets: assets}}
	coverRequest := httptest.NewRequest(http.MethodGet, "/media/private-pdf?variant=pdf-cover", nil)
	coverRequest = coverRequest.WithContext(auth.WithUser(coverRequest.Context(), &auth.User{ID: "editor", Role: auth.RoleAdmin, Active: true}))
	coverResponse := httptest.NewRecorder()
	handler.serve(coverResponse, coverRequest)
	if coverResponse.Code != http.StatusOK || coverResponse.Header().Get("Content-Type") != "image/png" || coverResponse.Header().Get("Content-Disposition") != "" {
		t.Fatalf("cover response = %d, headers=%v", coverResponse.Code, coverResponse.Header())
	}
	sourceRequest := httptest.NewRequest(http.MethodGet, "/media/private-pdf", nil)
	sourceRequest = sourceRequest.WithContext(auth.WithUser(sourceRequest.Context(), &auth.User{ID: "editor", Role: auth.RoleAdmin, Active: true}))
	sourceResponse := httptest.NewRecorder()
	handler.serve(sourceResponse, sourceRequest)
	if sourceResponse.Code != http.StatusOK || !strings.Contains(sourceResponse.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("source response = %d, disposition=%q", sourceResponse.Code, sourceResponse.Header().Get("Content-Disposition"))
	}
}

func minimalPDF() []byte {
	content := "0 0 1 rg 0 0 200 200 re f\n"
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Resources << >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(content), content),
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		offsets[i+1] = out.Len()
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&out, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes()
}
