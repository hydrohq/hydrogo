package socialsync

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

var (
	ErrSyncFailed = errors.New("upstream social sync failed")
)

type InstagramFetcher func(ctx context.Context) ([]SocialPost, error)
type GoogleReviewsFetcher func(ctx context.Context) ([]GoogleReview, error)

type SyncConfig struct {
	InstagramToken string
	GooglePlaceID  string
	GoogleAPIKey   string
	SyncInterval   time.Duration
}

type SyncService struct {
	Store          Store
	FetchInstagram InstagramFetcher
	FetchGoogle    GoogleReviewsFetcher
	Logger         *slog.Logger

	mu        sync.Mutex
	lastSync  time.Time
	lastError error
}

func NewSyncService(store Store, igFetcher InstagramFetcher, googleFetcher GoogleReviewsFetcher, logger *slog.Logger) *SyncService {
	if logger == nil {
		logger = slog.Default()
	}
	return &SyncService{
		Store:          store,
		FetchInstagram: igFetcher,
		FetchGoogle:    googleFetcher,
		Logger:         logger,
	}
}

// Sync runs a full synchronization pass. In case of upstream failures, existing cache is preserved.
func (s *SyncService) Sync(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error

	if s.FetchInstagram != nil {
		posts, err := s.FetchInstagram(ctx)
		if err != nil {
			s.Logger.WarnContext(ctx, "failed to fetch instagram posts; preserving existing cache", "error", err)
			errs = append(errs, err)
		} else if len(posts) > 0 {
			if saveErr := s.Store.SavePosts(ctx, posts); saveErr != nil {
				s.Logger.ErrorContext(ctx, "failed to persist instagram posts", "error", saveErr)
				errs = append(errs, saveErr)
			}
		}
	}

	if s.FetchGoogle != nil {
		reviews, err := s.FetchGoogle(ctx)
		if err != nil {
			s.Logger.WarnContext(ctx, "failed to fetch google reviews; preserving existing cache", "error", err)
			errs = append(errs, err)
		} else if len(reviews) > 0 {
			if saveErr := s.Store.SaveReviews(ctx, reviews); saveErr != nil {
				s.Logger.ErrorContext(ctx, "failed to persist google reviews", "error", saveErr)
				errs = append(errs, saveErr)
			}
		}
	}

	s.lastSync = time.Now().UTC()
	if len(errs) > 0 {
		s.lastError = errors.Join(errs...)
		return s.lastError
	}
	s.lastError = nil
	return nil
}

// PublicPosts retrieves cached social posts with sub-millisecond local latency.
func (s *SyncService) PublicPosts(ctx context.Context, limit int) ([]SocialPost, error) {
	if s.Store == nil {
		return nil, ErrStoreRequired
	}
	return s.Store.GetPosts(ctx, limit)
}

// PublicReviews retrieves 5-star or high-rated Google reviews from local cache.
func (s *SyncService) PublicReviews(ctx context.Context, minRating, limit int) ([]GoogleReview, error) {
	if s.Store == nil {
		return nil, ErrStoreRequired
	}
	return s.Store.GetReviews(ctx, minRating, limit)
}

func (s *SyncService) Status() (time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSync, s.lastError
}
