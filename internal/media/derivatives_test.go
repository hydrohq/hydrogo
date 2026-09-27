package media

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/deepteams/webp"
)

type recordingDerivativeRepository struct {
	mu      sync.Mutex
	created []Variant
	results []struct {
		id, status, detail string
		size               int64
	}
}

func (r *recordingDerivativeRepository) CreateProcessing(_ context.Context, _ database.Executor, variant Variant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.created = append(r.created, variant)
	return nil
}

func (r *recordingDerivativeRepository) MarkResult(_ context.Context, _ database.Executor, id, status string, size int64, detail string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results = append(r.results, struct {
		id, status, detail string
		size               int64
	}{id: id, status: status, size: size, detail: detail})
	return nil
}

func (r *recordingDerivativeRepository) GetReady(context.Context, database.Executor, string, string) (Variant, error) {
	return Variant{}, errors.New("not implemented in test repository")
}

type failingTransformer struct{ err error }

func (f failingTransformer) Transform(context.Context, Asset, string, string) (transformResult, error) {
	return transformResult{}, f.err
}

func TestOptimizerMarksEncoderFailure(t *testing.T) {
	const wantErr = "encoder unavailable"
	repo := &recordingDerivativeRepository{}
	optimizer := &Optimizer{
		transform: failingTransformer{err: errors.New(wantErr)},
		variants:  repo,
	}

	optimizer.process(Asset{ID: "asset-1", StorageKey: "asset.png", MimeType: "image/png"})

	if len(repo.created) != 1 || repo.created[0].Status != "processing" {
		t.Fatalf("created variants = %#v, want one processing variant", repo.created)
	}
	if len(repo.results) != 1 {
		t.Fatalf("result updates = %#v, want one failed update", repo.results)
	}
	if got := repo.results[0]; got.status != "failed" || got.size != 0 || !strings.Contains(got.detail, wantErr) {
		t.Fatalf("result update = %#v, want failed zero-size update containing %q", got, wantErr)
	}
}

func TestOptimizerQueueIsBoundedAndNonBlocking(t *testing.T) {
	optimizer := &Optimizer{queue: make(chan Asset, variantQueueCapacity)}
	for i := 0; i < variantQueueCapacity+3; i++ {
		optimizer.Enqueue(Asset{ID: string(rune('a' + i))})
	}
	if got := len(optimizer.queue); got != variantQueueCapacity {
		t.Fatalf("queue length = %d, want bounded capacity %d", got, variantQueueCapacity)
	}
}

func TestOptimizerCloseIsIdempotentAndRejectsLateEnqueue(t *testing.T) {
	optimizer := &Optimizer{queue: make(chan Asset, 1)}
	optimizer.Close()
	optimizer.Close()
	optimizer.Enqueue(Asset{ID: "late"})
	if got := len(optimizer.queue); got != 0 {
		t.Fatalf("closed optimizer accepted %d late assets", got)
	}
}

func TestVideoTransformFailsClosedWithoutEncoder(t *testing.T) {
	_, err := (ffmpegTransformer{storageDir: t.TempDir()}).Transform(context.Background(), Asset{StorageKey: "input.mp4"}, VariantVideo, "variant-1")
	if err == nil || !strings.Contains(err.Error(), "video encoder is not configured") {
		t.Fatalf("missing video encoder error = %v", err)
	}
}

func TestImageTransformProducesPureGoWebP(t *testing.T) {
	storage := t.TempDir()
	input := filepath.Join(storage, "input.png")
	file, err := os.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	if err := png.Encode(file, img); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	result, err := (ffmpegTransformer{storageDir: storage}).Transform(context.Background(), Asset{StorageKey: "input.png", MimeType: "image/png"}, VariantCard, "variant-webp")
	if err != nil {
		t.Fatalf("pure Go WebP transform failed: %v", err)
	}
	if result.Mime != "image/webp" || result.Key != "variant-webp.webp" || result.Size <= 0 {
		t.Fatalf("unexpected WebP result: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(storage, result.Key)); err != nil {
		t.Fatalf("WebP output missing: %v", err)
	}
	out, err := os.Open(filepath.Join(storage, result.Key))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := webp.Decode(out)
	_ = out.Close()
	if err != nil || decoded.Bounds().Dx() != 3 || decoded.Bounds().Dy() != 2 {
		t.Fatalf("WebP output decode = %v, error %v", decoded, err)
	}
}
