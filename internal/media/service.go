package media

import (
	"context"
	"errors"
	"io"
	"path/filepath"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

var ErrAssetRepositoryRequired = errors.New("asset repository is required")

type AssetRepository interface {
	Get(context.Context, database.Executor, string) (Asset, error)
	CreatePending(context.Context, database.Executor, Asset) error
	MarkReady(context.Context, database.Executor, string, string) error
}

func (s Service) Get(ctx context.Context, db database.Executor, id string) (Asset, error) {
	if s.Assets == nil {
		return Asset{}, ErrAssetRepositoryRequired
	}
	return s.Assets.Get(ctx, db, id)
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
	if s.Assets == nil {
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
	if s.Assets == nil {
		return ErrAssetRepositoryRequired
	}
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		return s.Assets.MarkReady(ctx, tx, id, visibility)
	})
}
