package socialsync

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSocialSyncStoreAndService(t *testing.T) {
	store := NewMemoryStore()

	mockPosts := []SocialPost{
		{
			ID:        "ig-1",
			Platform:  PlatformInstagram,
			MediaURL:  "https://example.com/photo.jpg",
			Permalink: "https://instagram.com/p/1",
			Caption:   "Hello HydroGo",
			PostedAt:  time.Now().Add(-1 * time.Hour),
			SyncedAt:  time.Now(),
		},
	}

	mockReviews := []GoogleReview{
		{
			ID:         "rev-1",
			AuthorName: "John Doe",
			Rating:     5,
			Text:       "Exceptional service and speed!",
			Published:  time.Now().Add(-24 * time.Hour),
			SyncedAt:   time.Now(),
		},
		{
			ID:         "rev-2",
			AuthorName: "Jane Smith",
			Rating:     3,
			Text:       "Good, but could improve.",
			Published:  time.Now().Add(-48 * time.Hour),
			SyncedAt:   time.Now(),
		},
	}

	fetchIG := func(ctx context.Context) ([]SocialPost, error) {
		return mockPosts, nil
	}
	fetchGoogle := func(ctx context.Context) ([]GoogleReview, error) {
		return mockReviews, nil
	}

	service := NewSyncService(store, fetchIG, fetchGoogle, nil)

	// Test 1: Initial Sync
	if err := service.Sync(context.Background()); err != nil {
		t.Fatalf("unexpected sync error: %v", err)
	}

	posts, err := service.PublicPosts(context.Background(), 10)
	if err != nil || len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d, err: %v", len(posts), err)
	}

	reviews, err := service.PublicReviews(context.Background(), 5, 10)
	if err != nil || len(reviews) != 1 {
		t.Fatalf("expected 1 5-star review, got %d, err: %v", len(reviews), err)
	}
	if reviews[0].AuthorName != "John Doe" {
		t.Fatalf("unexpected author: %s", reviews[0].AuthorName)
	}

	// Test 2: Upstream failure resilience (preserves cache)
	service.FetchInstagram = func(ctx context.Context) ([]SocialPost, error) {
		return nil, errors.New("rate limited")
	}
	_ = service.Sync(context.Background())

	cachedPosts, _ := service.PublicPosts(context.Background(), 10)
	if len(cachedPosts) != 1 {
		t.Fatalf("cache should be preserved on upstream error, got %d posts", len(cachedPosts))
	}
}

func TestSocialSyncHandler(t *testing.T) {
	store := NewMemoryStore()
	_ = store.SavePosts(context.Background(), []SocialPost{
		{ID: "p1", Platform: PlatformInstagram, Permalink: "https://instagr.am/1"},
	})
	_ = store.SaveReviews(context.Background(), []GoogleReview{
		{ID: "r1", AuthorName: "Alice", Rating: 5, Text: "Great"},
	})

	service := NewSyncService(store, nil, nil, nil)
	handler := NewHandler(service)

	// Test GET /api/v1/social/posts
	req := httptest.NewRequest(http.MethodGet, "/api/v1/social/posts?limit=5", nil)
	w := httptest.NewRecorder()
	handler.HandleGetPosts(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Test GET /api/v1/social/reviews
	reqRev := httptest.NewRequest(http.MethodGet, "/api/v1/social/reviews?min_rating=5", nil)
	wRev := httptest.NewRecorder()
	handler.HandleGetReviews(wRev, reqRev)
	if wRev.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wRev.Code)
	}
}

func TestSocialSyncWorker(t *testing.T) {
	store := NewMemoryStore()
	service := NewSyncService(store, nil, nil, nil)
	worker := NewWorker(service, 100*time.Millisecond, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	worker.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	worker.Stop()
}
