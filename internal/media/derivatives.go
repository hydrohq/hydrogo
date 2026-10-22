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
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	webp "github.com/deepteams/webp"
	pdfviewer "github.com/tucats/pdf-viewer"
	xdraw "golang.org/x/image/draw"
)

const (
	VariantCard          = "card"
	VariantPDFCover      = "pdf-cover"
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
	queue            chan optimizeJob
	workers          int
	transform        transformer
	variants         DerivativeRepository
	db               database.Executor
	storageDir       string
	blobStore        BlobStore
	defaultWatermark *WatermarkOptions
	wg               sync.WaitGroup
	lifecycle        sync.RWMutex
	closed           bool
}

type optimizeJob struct {
	asset     Asset
	watermark *WatermarkOptions
}

func NewOptimizer(storageDir string, workers int, variants DerivativeRepository, db database.Executor, ffmpegPath string) *Optimizer {
	return newOptimizer(storageDir, workers, variants, db, ffmpegPath, nil)
}

// NewOptimizerWithWatermark enables watermarking on generated image variants.
// Source objects remain unchanged.
func NewOptimizerWithWatermark(storageDir string, workers int, variants DerivativeRepository, db database.Executor, ffmpegPath string, options WatermarkOptions) *Optimizer {
	return newOptimizer(storageDir, workers, variants, db, ffmpegPath, &options)
}

func newOptimizer(storageDir string, workers int, variants DerivativeRepository, db database.Executor, ffmpegPath string, watermark *WatermarkOptions) *Optimizer {
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
	o := &Optimizer{queue: make(chan optimizeJob, variantQueueCapacity), workers: workers, variants: variants, db: db, storageDir: storageDir, defaultWatermark: watermark, transform: ffmpegTransformer{path: ffmpegPath, storageDir: storageDir}}
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

// SetBlobStore enables copying generated derivatives to the configured object
// store. It must be called before uploads are enqueued.
func (o *Optimizer) SetBlobStore(store BlobStore) {
	if o != nil {
		o.blobStore = store
	}
}
func (o *Optimizer) Enqueue(asset Asset) {
	if o == nil {
		return
	}
	o.EnqueueWithWatermark(asset, o.defaultWatermark)
}
func (o *Optimizer) EnqueueWithWatermark(asset Asset, watermark *WatermarkOptions) {
	if o == nil || o.queue == nil {
		return
	}
	o.lifecycle.RLock()
	defer o.lifecycle.RUnlock()
	if o.closed {
		return
	}
	select {
	case o.queue <- optimizeJob{asset: asset, watermark: watermark}:
	default:
	}
}
func (o *Optimizer) run() {
	defer o.wg.Done()
	for job := range o.queue {
		o.process(job)
	}
}
func (o *Optimizer) process(job optimizeJob) {
	asset := job.asset
	kind, mime := VariantCard, "image/webp"
	if strings.HasPrefix(asset.MimeType, "video/") {
		kind, mime = VariantVideo, "video/webm"
	} else if asset.MimeType == "application/pdf" {
		kind, mime = VariantPDFCover, "image/png"
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
	transformer := o.transform
	if imageTransformer, ok := transformer.(ffmpegTransformer); ok {
		imageTransformer.watermark = job.watermark
		transformer = imageTransformer
	}
	var result transformResult
	if asset.MimeType == "application/pdf" {
		result, err = renderPDFCover(ctx, filepath.Join(o.storageDir, filepath.Base(asset.StorageKey)), id)
	} else {
		result, err = transformer.Transform(ctx, asset, kind, id)
	}
	if err != nil {
		if o.variants != nil {
			_ = o.variants.MarkResult(ctx, o.db, id, "failed", 0, err.Error())
		}
		return
	}
	if o.blobStore != nil {
		key, keyErr := StorageKeyForContext(ctx, result.Key)
		file, openErr := os.Open(filepath.Join(o.storageDir, filepath.Base(result.Key)))
		if keyErr == nil && openErr == nil {
			putErr := o.blobStore.Put(ctx, key, file, result.Size, result.Mime)
			_ = file.Close()
			if putErr != nil {
				err = putErr
			}
		} else {
			if file != nil {
				_ = file.Close()
			}
			err = errors.Join(keyErr, openErr)
		}
		if err != nil {
			if o.variants != nil {
				_ = o.variants.MarkResult(ctx, o.db, id, "failed", 0, err.Error())
			}
			return
		}
	}
	if o.variants != nil {
		_ = o.variants.MarkResult(ctx, o.db, id, "ready", result.Size, result.Key)
	}
}

func renderPDFCover(ctx context.Context, input, id string) (result transformResult, err error) {
	defer func() {
		if recover() != nil {
			result = transformResult{}
			err = errors.New("PDF renderer rejected malformed input")
		}
	}()
	file, err := os.Open(input) // #nosec G304 -- path of a stored or temporary media file the pipeline passes in, never a client file name
	if err != nil {
		return transformResult{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 10<<20 {
		return transformResult{}, errors.New("PDF is outside the thumbnail size limit")
	}
	document, err := pdfviewer.Open(file, info.Size())
	if err != nil {
		return transformResult{}, fmt.Errorf("open PDF for cover rendering: %w", err)
	}
	defer document.Close()
	if document.PageCount() < 1 {
		return transformResult{}, errors.New("PDF has no pages")
	}
	page, err := document.Page(0)
	if err != nil {
		return transformResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	cover, err := page.Thumbnail(ctx, pdfviewer.ThumbnailOptions{MaxDimension: 320})
	if err != nil {
		return transformResult{}, fmt.Errorf("render PDF cover: %w", err)
	}
	out := filepath.Join(filepath.Dir(input), id+".png")
	tmp, err := os.CreateTemp(filepath.Dir(input), id+"-*.tmp")
	if err != nil {
		return transformResult{}, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := png.Encode(tmp, cover); err != nil {
		_ = tmp.Close()
		return transformResult{}, err
	}
	if err := tmp.Close(); err != nil {
		return transformResult{}, err
	}
	if err := os.Rename(tmpName, out); err != nil {
		return transformResult{}, err
	}
	resultInfo, err := os.Stat(out)
	if err != nil {
		return transformResult{}, err
	}
	return transformResult{Key: filepath.Base(out), Mime: "image/png", Size: resultInfo.Size()}, nil
}

type ffmpegTransformer struct {
	path, storageDir string
	watermark        *WatermarkOptions
}

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
	if err := exec.CommandContext(ctx, t.path, args...).Run(); err != nil { // #nosec G204 -- the transformer binary is the operator-configured ffmpeg; arguments are fixed flags plus stored paths, no shell
		return transformResult{}, fmt.Errorf("media transform failed: %w", err)
	}
	info, err := os.Stat(out)
	if err != nil || info.Size() <= 0 {
		return transformResult{}, errors.New("media transform produced no output")
	}
	return transformResult{Key: filepath.Base(out), Mime: map[string]string{".webp": "image/webp", ".webm": "video/webm"}[ext], Size: info.Size()}, nil
}

func (t ffmpegTransformer) transformImage(ctx context.Context, input, id string) (transformResult, error) {
	file, err := os.Open(input) // #nosec G304 -- path of a stored or temporary media file the pipeline passes in, never a client file name
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
	resized := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(resized, resized.Bounds(), decoded, decoded.Bounds(), xdraw.Over, nil)
	var dst image.Image = resized
	if t.watermark != nil {
		dst, err = ApplyWatermark(dst, *t.watermark)
		if err != nil {
			return transformResult{}, err
		}
	}
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
