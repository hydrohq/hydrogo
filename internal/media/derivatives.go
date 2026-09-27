package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	webp "github.com/deepteams/webp"
	xdraw "golang.org/x/image/draw"
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
	ctx, err := ContextForStorageKey(ctx, asset.StorageKey)
	if err != nil {
		return
	}
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
	input := filepath.Join(t.storageDir, filepath.Base(asset.StorageKey))
	if kind == VariantCard {
		return t.transformImage(ctx, input, id)
	}
	if strings.TrimSpace(t.path) == "" {
		return transformResult{}, errors.New("video encoder is not configured; set HYDRO_MEDIA_FFMPEG")
	}
	ext := ".webm"
	args := []string{"-hide_banner", "-loglevel", "error", "-threads", "1", "-y", "-i", input}
	args = append(args, "-t", "30", "-c:v", "libvpx-vp9", "-crf", "34", "-b:v", "0", "-deadline", "good", "-cpu-used", "4", "-c:a", "libopus", "-b:a", "96k")
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

func (t ffmpegTransformer) transformImage(ctx context.Context, input, id string) (transformResult, error) {
	file, err := os.Open(input)
	if err != nil {
		return transformResult{}, err
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return transformResult{}, fmt.Errorf("decode image header: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 40_000_000 {
		return transformResult{}, errors.New("image dimensions exceed derivative limit")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return transformResult{}, err
	}
	decoded, _, err := image.Decode(file)
	if err != nil {
		return transformResult{}, fmt.Errorf("decode image: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return transformResult{}, err
	}
	w, h := config.Width, config.Height
	if w > 1920 {
		h = h * 1920 / w
		w = 1920
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), decoded, decoded.Bounds(), xdraw.Over, nil)
	out := filepath.Join(t.storageDir, id+".webp")
	tmp, err := os.CreateTemp(t.storageDir, id+"-*.tmp")
	if err != nil {
		return transformResult{}, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := webp.Encode(tmp, dst, &webp.Options{Quality: 82}); err != nil {
		_ = tmp.Close()
		return transformResult{}, fmt.Errorf("encode WebP: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return transformResult{}, err
	}
	if err := os.Rename(tmpName, out); err != nil {
		return transformResult{}, err
	}
	info, err := os.Stat(out)
	if err != nil {
		return transformResult{}, err
	}
	return transformResult{Key: filepath.Base(out), Mime: "image/webp", Size: info.Size()}, nil
}
