package media

import (
	"context"
	"io"
	"path/filepath"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type AssetRepository interface {
	Get(context.Context, database.Executor, string) (Asset, error)
	CreatePending(context.Context, database.Executor, Asset) error
	MarkReady(context.Context, database.Executor, string, string) error
}

func (s Service) Get(ctx context.Context, db database.Executor, id string) (Asset, error) {
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
	return s.Transactions.Within(ctx, func(tx database.Executor) error {
		return s.Assets.MarkReady(ctx, tx, id, visibility)
	})
}
