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
	ID                 string
	Locale             string
	Slug               string
	Title              string
	Body               string
	Status             Status
	Version            int
	Revision           int
	PublishedRevision  *int
	ScheduleGeneration int64
	ScheduledAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
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
