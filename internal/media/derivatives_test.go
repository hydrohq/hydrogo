package media

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
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

func TestFFmpegTransformerFailsClosedWithoutEncoder(t *testing.T) {
	_, err := (ffmpegTransformer{storageDir: t.TempDir()}).Transform(context.Background(), Asset{StorageKey: "input.png"}, VariantCard, "variant-1")
	if err == nil || !strings.Contains(err.Error(), "ffmpeg is not configured") {
		t.Fatalf("missing encoder error = %v, want explicit configuration failure", err)
	}
}

func TestFFmpegTransformerProducesWebPWhenEncoderIsAvailable(t *testing.T) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is not installed")
	}
	storage := t.TempDir()
	input := filepath.Join(storage, "input.png")
	// A deterministic 1x1 PNG keeps this contract small and reproducible.
	pngBytes, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, pngBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := (ffmpegTransformer{path: ffmpegPath, storageDir: storage}).Transform(context.Background(), Asset{StorageKey: "input.png", MimeType: "image/png"}, VariantCard, "variant-webp")
	if err != nil {
		t.Fatalf("ffmpeg WebP transform failed: %v", err)
	}
	if result.Mime != "image/webp" || result.Key != "variant-webp.webp" || result.Size <= 0 {
		t.Fatalf("unexpected WebP result: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(storage, result.Key)); err != nil {
		t.Fatalf("WebP output missing: %v", err)
	}
}
