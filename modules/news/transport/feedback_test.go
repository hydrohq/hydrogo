package transport

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ardaozyildirim/hydrogo/internal/analytics"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/modules/news"
)

type feedbackNewsRepo struct{ news.Repository }

func (feedbackNewsRepo) GetPublishedByRoute(_ context.Context, _ database.Executor, locale, slug string) (news.Item, error) {
	return news.Item{ID: "article-1", Locale: locale, Slug: slug, Title: "A story", Body: "Some content"}, nil
}

type feedbackNewsStore struct{}

func (feedbackNewsStore) FeedbackTotals(context.Context, string, string, string) (analytics.FeedbackTotals, error) {
	return analytics.FeedbackTotals{Up: 4, Down: 1}, nil
}
func (feedbackNewsStore) FeedbackCounts(context.Context, string, string, []string) (map[string]analytics.FeedbackTotals, error) {
	return map[string]analytics.FeedbackTotals{}, nil
}

func TestPublishedNewsHTMLIncludesHelpfulFeedbackComponent(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "news-feedback.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	h := Handler{DB: db, Service: news.Service{Repository: feedbackNewsRepo{}}, Feedback: feedbackNewsStore{}}
	r := httptest.NewRequest("GET", "/public/news/tr/story/html", nil)
	r.SetPathValue("locale", "tr")
	r.SetPathValue("slug", "story")
	w := httptest.NewRecorder()
	h.publicGetHTML(w, r)
	for _, want := range []string{`href="/designsystem/static/designsystem.css"`, `hx-post="/api/v1/feedback"`, `data-feedback-type="news"`, `4 kişi faydalı buldu · 1 kişi faydalı bulmadı`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Errorf("news feedback response missing %q", want)
		}
	}
}
