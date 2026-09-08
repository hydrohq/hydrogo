package news

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidItem   = errors.New("invalid news item")
	ErrConflict      = errors.New("news item conflict")
	ErrScheduleStale = errors.New("news schedule is stale")
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
	item.Locale = strings.ToLower(strings.TrimSpace(item.Locale))
	item.Slug = strings.Trim(strings.ToLower(strings.TrimSpace(item.Slug)), "/")
	item.Title = strings.TrimSpace(item.Title)
	if item.ID == "" || item.Locale == "" || item.Slug == "" || item.Title == "" || item.Status != StatusDraft {
		return ErrInvalidItem
	}
	return nil
}
