package trash

import (
	"context"
	"errors"
	"time"
)

const Retention = 30 * 24 * time.Hour

var ErrNotFound = errors.New("trash record not found")

type Kind string

const (
	Pages       Kind = "pages"
	News        Kind = "news"
	Submissions Kind = "submissions"
	Media       Kind = "media"
)

type Entry struct {
	Kind      Kind
	ID        string
	Label     string
	DeletedAt time.Time
	Trashed   bool
}

type Repository interface {
	List(context.Context, int) ([]Entry, error)
	Move(context.Context, Kind, string, time.Time) error
	Restore(context.Context, Kind, string, time.Time) error
}

type Service struct{ Repository Repository }

func (s Service) List(ctx context.Context, limit int) ([]Entry, error) {
	if s.Repository == nil {
		return nil, errors.New("trash repository is required")
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	return s.Repository.List(ctx, limit)
}

func (s Service) Move(ctx context.Context, kind Kind, id string, now time.Time) error {
	if s.Repository == nil || !valid(kind) || id == "" || len(id) > 128 {
		return ErrNotFound
	}
	return s.Repository.Move(ctx, kind, id, now.UTC())
}

func (s Service) Restore(ctx context.Context, kind Kind, id string, now time.Time) error {
	if s.Repository == nil || !valid(kind) || id == "" || len(id) > 128 {
		return ErrNotFound
	}
	return s.Repository.Restore(ctx, kind, id, now.UTC().Add(-Retention))
}

func valid(kind Kind) bool {
	switch kind {
	case Pages, News, Submissions, Media:
		return true
	default:
		return false
	}
}
