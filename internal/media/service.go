package media

import (
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

var ErrAssetRepositoryRequired = errors.New("asset repository is required")
var ErrAssetListingUnsupported = errors.New("asset repository does not support listing")
var ErrAssetVisibilityUnsupported = errors.New("asset repository does not support visibility updates")
var ErrQuarantineCompletion = errors.New("quarantine upload completion is invalid")

type AssetRepository interface {
	Get(context.Context, database.Executor, string) (Asset, error)
	CreatePending(context.Context, database.Executor, Asset) error
	MarkReady(context.Context, database.Executor, string, string) error
}

// AssetListRepository is optional so existing storage adapters and focused
// upload tests do not gain a read-heavy admin dependency.
type AssetListRepository interface {
	List(context.Context, database.Executor, int) ([]Asset, error)
}

// AssetVisibilityRepository is deliberately separate from the upload state
// transition: ready assets can be reclassified without pretending to upload
// them again.
type AssetVisibilityRepository interface {
	SetVisibility(context.Context, database.Executor, string, string) error
}

type VerifiedAssetRepository interface {
	MarkReadyVerified(context.Context, database.Executor, string, string, string) error
}

type AssetMetadataRepository interface {
	SetMetadata(context.Context, database.Executor, string, AssetMetadata) error
}

type AssetMetadata struct {
	Focal    FocalPoint `json:"focal_point"`
	FolderID string     `json:"folder_id,omitempty"`
	Tags     []string   `json:"tags,omitempty"`
}

// AdminSettings are tenant-scoped controls for media derivatives and delivery.
type AdminSettings struct {
	HotlinkProtection   bool     `json:"hotlink_protection"`
	AllowedRefererHosts []string `json:"allowed_referer_hosts"`
	WatermarkEnabled    bool     `json:"watermark_enabled"`
	WatermarkText       string   `json:"watermark_text"`
	WatermarkPosition   string   `json:"watermark_position"`
	WatermarkOpacity    float64  `json:"watermark_opacity"`
	WatermarkMargin     int      `json:"watermark_margin"`
	WatermarkLogoID     string   `json:"watermark_logo_id,omitempty"`
	WatermarkFolders    []string `json:"watermark_folders,omitempty"`
}

type AdminSettingsRepository interface {
	LoadAdminSettings(context.Context, database.Executor) (AdminSettings, bool, error)
	SaveAdminSettings(context.Context, database.Executor, AdminSettings) error
}

func (s Service) SetMetadata(ctx context.Context, id string, metadata AssetMetadata) error {
	if guards.IsNil(s.Assets) {
		return ErrAssetRepositoryRequired
	}
	if clamp01(metadata.Focal.X) != metadata.Focal.X || clamp01(metadata.Focal.Y) != metadata.Focal.Y || len(metadata.Tags) > 32 {
		return ErrInvalidFolder
	}
	for _, tag := range metadata.Tags {
		if strings.TrimSpace(tag) == "" || len(tag) > 64 {
			return ErrInvalidFolder
		}
	}
	repo, ok := s.Assets.(AssetMetadataRepository)
	if !ok {
		return errors.New("media metadata updates are unsupported")
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error { return repo.SetMetadata(ctx, tx, id, metadata) })
}

func (s Service) CreateFolder(ctx context.Context, folder Folder) error {
	repo, ok := s.Assets.(FolderRepository)
	if !ok {
		return errors.New("media folders are unsupported")
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error { return repo.CreateFolder(ctx, tx, folder) })
}

func (s Service) ListFolders(ctx context.Context, db database.Executor) ([]Folder, error) {
	repo, ok := s.Assets.(FolderRepository)
	if !ok {
		return nil, errors.New("media folders are unsupported")
	}
	return repo.ListFolders(ctx, db)
}

func (s Service) Get(ctx context.Context, db database.Executor, id string) (Asset, error) {
	if guards.IsNil(s.Assets) {
		return Asset{}, ErrAssetRepositoryRequired
	}
	return s.Assets.Get(ctx, db, id)
}

// PublicURL returns a provider-backed public URL when the configured blob
// store exposes that optional capability. Callers must keep their local route
// fallback when no public provider is configured.
func (s Service) PublicURL(key string) (string, error) {
	provider, ok := s.BlobStore.(PublicURLProvider)
	if !ok {
		return "", errors.New("public object URL is not configured")
	}
	return provider.PublicURL(key)
}

func (s Service) List(ctx context.Context, db database.Executor, limit int) ([]Asset, error) {
	if guards.IsNil(s.Assets) {
		return nil, ErrAssetRepositoryRequired
	}
	repository, ok := s.Assets.(AssetListRepository)
	if !ok {
		return nil, ErrAssetListingUnsupported
	}
	return repository.List(ctx, db, limit)
}

// Service coordinates immutable file creation with the pending/ready metadata state.
// A failed metadata write leaves an unreferenced file for the storage sweeper.
type Service struct {
	Transactions database.Transactor
	Uploader     *Uploader
	// BlobStore is optional. When set, immutable objects are written to and
	// served from the configured provider-neutral object store.
	BlobStore        BlobStore
	Assets           AssetRepository
	Optimizer        *Optimizer
	AdminSettings    AdminSettingsRepository
	DefaultWatermark *WatermarkOptions
}

func (s Service) Upload(ctx context.Context, reader io.Reader, filename string) (*UploadResult, error) {
	if s.Uploader == nil {
		return nil, ErrUploaderRequired
	}
	if guards.IsNil(s.Assets) {
		return nil, ErrAssetRepositoryRequired
	}
	result, err := s.Uploader.SaveContext(ctx, reader, filename)
	if err != nil {
		return nil, err
	}
	return s.storeUpload(ctx, result)
}

// UploadPrivateDocument validates and stores an employment-form attachment as
// private media with the document-specific size cap.
func (s Service) UploadPrivateDocument(ctx context.Context, reader io.Reader, filename string) (*UploadResult, error) {
	if s.Uploader == nil {
		return nil, ErrUploaderRequired
	}
	if guards.IsNil(s.Assets) {
		return nil, ErrAssetRepositoryRequired
	}
	result, err := s.Uploader.SaveDocumentContext(ctx, reader, filename, 10<<20)
	if err != nil {
		return nil, err
	}
	stored, err := s.storeUpload(ctx, result)
	if err != nil {
		return nil, err
	}
	if err := s.MarkReady(ctx, stored.ID, VisibilityPrivate); err != nil {
		return nil, err
	}
	return stored, nil
}

func (s Service) storeUpload(ctx context.Context, result *UploadResult) (*UploadResult, error) {
	storageKey, err := StorageKeyForContext(ctx, filepath.Base(result.StoragePath))
	if err != nil {
		return nil, err
	}
	asset := Asset{
		ID:               result.ID,
		StorageKey:       storageKey,
		OriginalFilename: result.Filename,
		MimeType:         result.MimeType,
		SizeBytes:        result.SizeBytes,
		Width:            result.Width,
		Height:           result.Height,
		BlurHash:         result.BlurHash,
		FocalX:           0.5,
		FocalY:           0.5,
		DominantColors:   result.DominantColors,
	}
	if err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		return s.Assets.CreatePending(ctx, tx, asset)
	}); err != nil {
		return nil, err
	}
	if s.BlobStore != nil {
		file, openErr := os.Open(result.StoragePath)
		if openErr != nil {
			return nil, openErr
		}
		defer file.Close()
		if err := s.BlobStore.Put(ctx, storageKey, file, result.SizeBytes, result.MimeType); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s Service) MarkReady(ctx context.Context, id, visibility string) error {
	if guards.IsNil(s.Assets) {
		return ErrAssetRepositoryRequired
	}
	var asset Asset
	var watermark *WatermarkOptions
	if s.Optimizer != nil {
		var err error
		asset, err = s.Get(ctx, s.Optimizer.db, id)
		if err != nil {
			return err
		}
		watermark = s.DefaultWatermark
		if s.AdminSettings != nil {
			settings, found, settingsErr := s.AdminSettings.LoadAdminSettings(ctx, s.Optimizer.db)
			if settingsErr != nil {
				return settingsErr
			}
			if found {
				watermark = nil
				folderSelected := len(settings.WatermarkFolders) == 0
				for _, folderID := range settings.WatermarkFolders {
					if folderID == asset.FolderID {
						folderSelected = true
						break
					}
				}
				if strings.HasPrefix(asset.MimeType, "image/") && settings.WatermarkEnabled && (settings.WatermarkText != "" || settings.WatermarkLogoID != "") && folderSelected {
					opacity := settings.WatermarkOpacity
					if opacity == 0 {
						opacity = .6
					}
					watermark = &WatermarkOptions{Text: settings.WatermarkText, Position: settings.WatermarkPosition, Opacity: opacity, Margin: settings.WatermarkMargin}
					if settings.WatermarkLogoID != "" {
						watermark.Logo, err = s.loadWatermarkLogo(ctx, settings.WatermarkLogoID)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		return s.Assets.MarkReady(ctx, tx, id, visibility)
	})
	if err == nil && s.Optimizer != nil {
		s.Optimizer.EnqueueWithWatermark(asset, watermark)
	}
	return err
}

func (s Service) loadWatermarkLogo(ctx context.Context, id string) (image.Image, error) {
	asset, err := s.Get(ctx, s.Optimizer.db, id)
	if err != nil || asset.Status != "ready" || asset.MimeType != "image/png" || asset.SizeBytes <= 0 || asset.SizeBytes > 2<<20 {
		return nil, errors.New("configured watermark logo is unavailable")
	}
	var reader io.ReadCloser
	if s.BlobStore != nil {
		reader, err = s.BlobStore.Get(ctx, asset.StorageKey)
	} else if s.Uploader != nil {
		reader, err = os.Open(filepath.Join(s.Uploader.StorageDir, filepath.Base(asset.StorageKey)))
	}
	if err != nil || reader == nil {
		return nil, errors.New("configured watermark logo is unavailable")
	}
	defer reader.Close()
	logo, err := png.Decode(io.LimitReader(reader, 2<<20))
	if err != nil {
		return nil, errors.New("configured watermark logo is invalid")
	}
	bounds := logo.Bounds()
	if bounds.Dx() < 1 || bounds.Dy() < 1 || int64(bounds.Dx())*int64(bounds.Dy()) > 16_000_000 {
		return nil, errors.New("configured watermark logo dimensions exceed limit")
	}
	return logo, nil
}

// CompleteQuarantineUpload verifies the provider object before promoting a
// pending asset. The storage key is supplied by the server-issued intent and
// must remain inside quarantine until verification succeeds.
func (s Service) CompleteQuarantineUpload(ctx context.Context, id, objectKey string, expectedSize int64, expectedSHA256, visibility string) error {
	if s.BlobStore == nil || guards.IsNil(s.Assets) {
		return ErrQuarantineCompletion
	}
	if _, parts, err := validateObjectKey(objectKey); err != nil || !isQuarantineKey(parts) {
		return ErrQuarantineCompletion
	}
	if err := VerifyObject(ctx, s.BlobStore, objectKey, expectedSize, expectedSHA256); err != nil {
		return err
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		if verified, ok := s.Assets.(VerifiedAssetRepository); ok {
			return verified.MarkReadyVerified(ctx, tx, id, objectKey, visibility)
		}
		return s.Assets.MarkReady(ctx, tx, id, visibility)
	})
}

func (s Service) SetVisibility(ctx context.Context, id, visibility string) error {
	if guards.IsNil(s.Assets) {
		return ErrAssetRepositoryRequired
	}
	repository, ok := s.Assets.(AssetVisibilityRepository)
	if !ok {
		return ErrAssetVisibilityUnsupported
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		return repository.SetVisibility(ctx, tx, id, visibility)
	})
}
