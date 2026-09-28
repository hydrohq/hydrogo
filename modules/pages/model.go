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
	ID                    string    `json:"id"`
	Locale                string    `json:"locale"`
	Slug                  string    `json:"slug"`
	Title                 string    `json:"title"`
	Body                  string    `json:"body"`
	Blocks                []Block   `json:"blocks"`
	Tags                  []string  `json:"tags,omitempty"`
	Status                Status    `json:"status"`
	Version               int       `json:"version"`
	Revision              int       `json:"revision"`
	PublishedRevision     *int      `json:"published_revision,omitempty"`
	PublicationGeneration int64     `json:"publication_generation"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type RevisionSnapshot struct {
	ID         string
	PageID     string
	Revision   int
	Title      string
	Body       string
	Blocks     []Block
	CreatedAt  time.Time
	ActorID    string
	ActorEmail string
}
