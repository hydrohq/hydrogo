package adminstudio

import "time"

// ResourceAccess describes basic CRUD capabilities for a resource or record.
// An empty record ID asks for collection-level create access.
type ResourceAccess struct {
	Create bool `json:"create"`
	Update bool `json:"update"`
	Delete bool `json:"delete"`
}

type LocaleOptions struct {
	Default string   `json:"default"`
	Allowed []string `json:"allowed"`
}

type RevisionSummary struct {
	Version   int64     `json:"version"`
	State     string    `json:"state"`
	Actor     string    `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
}

type RevisionSnapshot struct {
	RevisionSummary
	Fields map[string]string `json:"fields"`
}

// DraftDocument is a working copy. PublishedVersion is checked independently
// from the draft version so a concurrent publish cannot be silently replaced.
type DraftDocument struct {
	ID               string           `json:"id"`
	Version          int64            `json:"version"`
	PublishedVersion int64            `json:"published_version"`
	State            PublicationState `json:"state"`
	UpdatedAt        time.Time        `json:"updated_at"`
	Fields           map[string]any   `json:"fields"`
	Exists           bool             `json:"exists"`
}
