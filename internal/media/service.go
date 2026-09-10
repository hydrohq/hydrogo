package media

import (
	"context"
	"errors"
	"io"
	"path/filepath"

	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/guards"
)

var ErrAssetRepositoryRequired = errors.New("asset repository is required")
var ErrAssetListingUnsupported = errors.New("asset repository does not support listing")

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

func (s Service) Get(ctx context.Context, db database.Executor, id string) (Asset, error) {
	if guards.IsNil(s.Assets) {
		return Asset{}, ErrAssetRepositoryRequired
	}
	return s.Assets.Get(ctx, db, id)
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
	Assets       AssetRepository
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
	asset := Asset{
		ID:               result.ID,
		StorageKey:       filepath.Base(result.StoragePath),
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
	return result, nil
}

func (s Service) MarkReady(ctx context.Context, id, visibility string) error {
	if guards.IsNil(s.Assets) {
		return ErrAssetRepositoryRequired
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		return s.Assets.MarkReady(ctx, tx, id, visibility)
	})
}
