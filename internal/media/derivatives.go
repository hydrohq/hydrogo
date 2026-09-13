package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	VariantCard          = "card"
	VariantVideo         = "video"
	variantQueueCapacity = 8
)

type Variant struct {
	ID, MediaID, Kind, StorageKey, MimeType, Status, ErrorMessage string
	SizeBytes, Width, Height, DurationMs                          int64
}

// VariantStore is kept separate from AssetRepository so existing adapters can
// adopt derivative support without changing the upload contract.
type DerivativeRepository interface {
	CreateProcessing(context.Context, database.Executor, Variant) error
	MarkResult(context.Context, database.Executor, string, string, int64, string) error
	GetReady(context.Context, database.Executor, string, string) (Variant, error)
}

type transformResult struct {
	Key, Mime string
	Size      int64
}
type transformer interface {
	Transform(context.Context, Asset, string, string) (transformResult, error)
}

// Optimizer is intentionally small and bounded. Enqueue never allocates per
// upload and a full queue cannot block the upload or grow process memory.
type Optimizer struct {
	queue      chan Asset
	workers    int
	transform  transformer
	variants   DerivativeRepository
	db         database.Executor
	storageDir string
	wg         sync.WaitGroup
	lifecycle  sync.RWMutex
	closed     bool
}

func NewOptimizer(storageDir string, workers int, variants DerivativeRepository, db database.Executor, ffmpegPath string) *Optimizer {
	if workers < 1 {
		workers = 1
	}
	if workers > 4 {
		workers = 4
	}
	if strings.TrimSpace(ffmpegPath) == "" {
		if discovered, err := exec.LookPath("ffmpeg"); err == nil {
			ffmpegPath = discovered
		}
	}
	o := &Optimizer{queue: make(chan Asset, variantQueueCapacity), workers: workers, variants: variants, db: db, storageDir: storageDir, transform: ffmpegTransformer{path: ffmpegPath, storageDir: storageDir}}
	for i := 0; i < workers; i++ {
		o.wg.Add(1)
		go o.run()
	}
	return o
}
func (o *Optimizer) Close() {
	if o == nil {
		return
	}
	o.lifecycle.Lock()
	if o.closed {
		o.lifecycle.Unlock()
		return
	}
	o.closed = true
	close(o.queue)
	o.lifecycle.Unlock()
	o.wg.Wait()
}
func (o *Optimizer) Enqueue(asset Asset) {
	if o == nil || o.queue == nil {
		return
	}
	o.lifecycle.RLock()
	defer o.lifecycle.RUnlock()
	if o.closed {
		return
	}
	select {
	case o.queue <- asset:
	default:
	}
}
func (o *Optimizer) run() {
	defer o.wg.Done()
	for asset := range o.queue {
		o.process(asset)
	}
}
func (o *Optimizer) process(asset Asset) {
	kind, mime := VariantCard, "image/webp"
	if strings.HasPrefix(asset.MimeType, "video/") {
		kind, mime = VariantVideo, "video/webm"
	} else if !strings.HasPrefix(asset.MimeType, "image/") {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	idBytes := sha256.Sum256([]byte(asset.ID + ":" + kind))
	id := hex.EncodeToString(idBytes[:])
	if o.variants != nil {
		_ = o.variants.CreateProcessing(ctx, o.db, Variant{ID: id, MediaID: asset.ID, Kind: kind, MimeType: mime, Status: "processing"})
	}
	result, err := o.transform.Transform(ctx, asset, kind, id)
	if err != nil {
		if o.variants != nil {
			_ = o.variants.MarkResult(ctx, o.db, id, "failed", 0, err.Error())
		}
		return
	}
	if o.variants != nil {
		_ = o.variants.MarkResult(ctx, o.db, id, "ready", result.Size, result.Key)
	}
}

type ffmpegTransformer struct{ path, storageDir string }

func (t ffmpegTransformer) Transform(ctx context.Context, asset Asset, kind, id string) (transformResult, error) {
	if strings.TrimSpace(t.path) == "" {
		return transformResult{}, errors.New("ffmpeg is not configured; set HYDRO_MEDIA_FFMPEG")
	}
	input := filepath.Join(t.storageDir, filepath.Base(asset.StorageKey))
	ext := ".webp"
	args := []string{"-hide_banner", "-loglevel", "error", "-threads", "1", "-y", "-i", input}
	if kind == VariantCard {
		args = append(args, "-frames:v", "1", "-vf", "scale='min(1920,iw)':-2", "-c:v", "libwebp", "-quality", "82")
	} else {
		ext = ".webm"
		args = append(args, "-t", "30", "-c:v", "libvpx-vp9", "-crf", "34", "-b:v", "0", "-deadline", "good", "-cpu-used", "4", "-c:a", "libopus", "-b:a", "96k")
	}
	out := filepath.Join(t.storageDir, id+ext)
	args = append(args, out)
	if err := exec.CommandContext(ctx, t.path, args...).Run(); err != nil {
		return transformResult{}, fmt.Errorf("media transform failed: %w", err)
	}
	info, err := os.Stat(out)
	if err != nil || info.Size() <= 0 {
		return transformResult{}, errors.New("media transform produced no output")
	}
	return transformResult{Key: filepath.Base(out), Mime: map[string]string{".webp": "image/webp", ".webm": "video/webm"}[ext], Size: info.Size()}, nil
}
