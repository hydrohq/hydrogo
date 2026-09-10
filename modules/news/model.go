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
	Status             Status     `json:"status"`
	Version            int        `json:"version"`
	Revision           int        `json:"revision"`
	PublishedRevision  *int       `json:"published_revision,omitempty"`
	ScheduleGeneration int64      `json:"schedule_generation"`
	ScheduledAt        *time.Time `json:"scheduled_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func Validate(item Item) error {
	if err := Canonicalize(&item); err != nil {
		return ErrInvalidItem
	}
	item.Title = strings.TrimSpace(item.Title)
	if item.ID == "" || item.Locale == "" || item.Slug == "" || item.Title == "" || item.Status != StatusDraft {
		return ErrInvalidItem
	}
	return nil
}

func Canonicalize(item *Item) error {
	if item == nil {
		return canonical.ErrInvalidPath
	}
	item.Locale = canonical.Locale(item.Locale)
	var err error
	item.Slug, err = canonical.Path(item.Slug)
	return err
}
