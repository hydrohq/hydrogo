package media

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

var ErrAssetRepositoryRequired = errors.New("asset repository is required")
var ErrAssetListingUnsupported = errors.New("asset repository does not support listing")
var ErrAssetVisibilityUnsupported = errors.New("asset repository does not support visibility updates")

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
	BlobStore BlobStore
	Assets    AssetRepository
	Optimizer *Optimizer
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
	err := s.Transactions.Within(ctx, func(tx database.Executor) error {
		return s.Assets.MarkReady(ctx, tx, id, visibility)
	})
	if err == nil && s.Optimizer != nil {
		if asset, getErr := s.Get(ctx, s.Optimizer.db, id); getErr == nil {
			s.Optimizer.Enqueue(asset)
		}
	}
	return err
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
