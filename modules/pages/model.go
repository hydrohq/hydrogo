package pages

import (
	"errors"
	"time"
)

var (
	ErrConflict      = errors.New("page update conflict")
	ErrRouteConflict = errors.New("published route conflict")
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
)

type Page struct {
	ID                    string
	Locale                string
	Slug                  string
	Title                 string
	Body                  string
	Blocks                []Block
	Status                Status
	Version               int
	Revision              int
	PublishedRevision     *int
	PublicationGeneration int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
