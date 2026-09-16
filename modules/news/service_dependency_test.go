package news

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

type localeFallbackRepository struct {
	failingRepository
	seen []string
}

func (r *localeFallbackRepository) GetPublishedByRoute(_ context.Context, _ database.Executor, locale, _ string) (Item, error) {
	r.seen = append(r.seen, locale)
	if locale == "en" {
		return Item{ID: "item-en", Locale: "en"}, nil
	}
	return Item{}, sql.ErrNoRows
}

func TestGetPublishedByRouteUsesConfiguredFallback(t *testing.T) {
	repo := &localeFallbackRepository{}
	service := Service{Repository: repo, SupportedLocales: []string{"tr", "en"}, DefaultLocale: "en"}
	item, err := service.GetPublishedByRoute(context.Background(), nil, "tr-TR", "home")
	if err != nil || item.ID != "item-en" {
		t.Fatalf("fallback item = %+v/%v", item, err)
	}
	if len(repo.seen) != 2 || repo.seen[0] != "tr" || repo.seen[1] != "en" {
		t.Fatalf("fallback chain = %#v", repo.seen)
	}
}

func TestServiceRejectsMissingRepository(t *testing.T) {
	err := (Service{}).Publish(context.Background(), "item-1", 1)
	if !errors.Is(err, ErrRepositoryRequired) {
		t.Fatalf("expected ErrRepositoryRequired, got %v", err)
	}
}
