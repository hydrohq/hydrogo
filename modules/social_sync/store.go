package socialsync

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrStoreRequired = errors.New("storage backend is required")
)

type Store interface {
	SavePosts(ctx context.Context, posts []SocialPost) error
	GetPosts(ctx context.Context, limit int) ([]SocialPost, error)
	SaveReviews(ctx context.Context, reviews []GoogleReview) error
	GetReviews(ctx context.Context, minRating int, limit int) ([]GoogleReview, error)
}

// MemoryStore provides a resilient, in-memory local cache with sub-millisecond retrieval.
type MemoryStore struct {
	mu      sync.RWMutex
	posts   []SocialPost
	reviews []GoogleReview
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		posts:   make([]SocialPost, 0),
		reviews: make([]GoogleReview, 0),
	}
}

func (s *MemoryStore) SavePosts(ctx context.Context, posts []SocialPost) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range posts {
		if err := p.Validate(); err != nil {
			return err
		}
	}
	s.posts = append([]SocialPost(nil), posts...)
	return nil
}

func (s *MemoryStore) GetPosts(ctx context.Context, limit int) ([]SocialPost, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.posts) {
		limit = len(s.posts)
	}
	res := make([]SocialPost, limit)
	copy(res, s.posts[:limit])
	return res, nil
}

func (s *MemoryStore) SaveReviews(ctx context.Context, reviews []GoogleReview) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range reviews {
		if err := r.Validate(); err != nil {
			return err
		}
	}
	s.reviews = append([]GoogleReview(nil), reviews...)
	return nil
}

func (s *MemoryStore) GetReviews(ctx context.Context, minRating int, limit int) ([]GoogleReview, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	filtered := make([]GoogleReview, 0, len(s.reviews))
	for _, r := range s.reviews {
		if r.Rating >= minRating {
			filtered = append(filtered, r)
		}
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}
