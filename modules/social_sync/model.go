package socialsync

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidPost   = errors.New("invalid social media post")
	ErrInvalidReview = errors.New("invalid google review")
)

type Platform string

const (
	PlatformInstagram Platform = "instagram"
	PlatformGoogle    Platform = "google"
)

type SocialPost struct {
	ID        string    `json:"id"`
	Platform  Platform  `json:"platform"`
	MediaURL  string    `json:"media_url"`
	Permalink string    `json:"permalink"`
	Caption   string    `json:"caption"`
	PostedAt  time.Time `json:"posted_at"`
	SyncedAt  time.Time `json:"synced_at"`
}

func (p SocialPost) Validate() error {
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.Permalink) == "" {
		return ErrInvalidPost
	}
	if p.Platform != PlatformInstagram {
		return ErrInvalidPost
	}
	return nil
}

type GoogleReview struct {
	ID         string    `json:"id"`
	AuthorName string    `json:"author_name"`
	AuthorURL  string    `json:"author_url,omitempty"`
	Rating     int       `json:"rating"`
	Text       string    `json:"text"`
	Published  time.Time `json:"published"`
	SyncedAt   time.Time `json:"synced_at"`
}

func (r GoogleReview) Validate() error {
	if strings.TrimSpace(r.ID) == "" || strings.TrimSpace(r.AuthorName) == "" {
		return ErrInvalidReview
	}
	if r.Rating < 1 || r.Rating > 5 {
		return ErrInvalidReview
	}
	return nil
}
