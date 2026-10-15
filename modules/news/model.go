package news

import (
	"errors"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/canonical"
)

var (
	ErrInvalidItem        = errors.New("invalid news item")
	ErrConflict           = errors.New("news item conflict")
	ErrScheduleStale      = errors.New("news schedule is stale")
	ErrRepositoryRequired = errors.New("news repository is required")
	// ErrTaxonomyTermInUse and ErrTaxonomyTermHasChildren refuse to delete a
	// term that items or child terms still reference.
	ErrTaxonomyTermInUse       = errors.New("taxonomy term is in use")
	ErrTaxonomyTermHasChildren = errors.New("taxonomy term has children")
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusScheduled Status = "scheduled"
	StatusPublished Status = "published"
	StatusRetracted Status = "retracted"
)

type Item struct {
	ID                 string     `json:"id"`
	Locale             string     `json:"locale"`
	Slug               string     `json:"slug"`
	Title              string     `json:"title"`
	Body               string     `json:"body"`
	CoverMediaID       string     `json:"cover_media_id,omitempty"`
	Excerpt            string     `json:"excerpt,omitempty"`
	KeyTakeaways       string     `json:"key_takeaways,omitempty"`
	Category           string     `json:"category,omitempty"`
	AuthorName         string     `json:"author_name,omitempty"`
	Featured           bool       `json:"featured"`
	Tags               string     `json:"tags,omitempty"`
	SEOTitle           string     `json:"seo_title,omitempty"`
	SEODescription     string     `json:"seo_description,omitempty"`
	Status             Status     `json:"status"`
	Version            int        `json:"version"`
	Revision           int        `json:"revision"`
	PublishedRevision  *int       `json:"published_revision,omitempty"`
	ScheduleGeneration int64      `json:"schedule_generation"`
	ScheduledAt        *time.Time `json:"scheduled_at,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func Validate(item Item) error {
	if err := Canonicalize(&item); err != nil {
		return ErrInvalidItem
	}
	item.Title = strings.TrimSpace(item.Title)
	if item.ID == "" || len(item.ID) > 128 || item.Locale == "" || len(item.Locale) > 16 || item.Slug == "" || len(item.Slug) > 256 || item.Title == "" || len(item.Title) > 256 || len(item.Excerpt) > 4000 || len(item.KeyTakeaways) > 12000 || len(item.Category) > 128 || len(item.AuthorName) > 256 || len(item.Tags) > 2000 || len(item.SEOTitle) > 256 || len(item.SEODescription) > 1000 || item.Status != StatusDraft {
		return ErrInvalidItem
	}
	if item.ExpiresAt != nil && item.ExpiresAt.IsZero() {
		return ErrInvalidItem
	}
	return nil
}

func Canonicalize(item *Item) error {
	if item == nil {
		return canonical.ErrInvalidPath
	}
	item.Locale = canonical.Locale(item.Locale)
	item.Slug = canonical.Slugify(item.Slug)
	if item.Slug == "" {
		item.Slug = canonical.Slugify(item.Title)
	}
	var err error
	item.Slug, err = canonical.Path(item.Slug)
	return err
}
