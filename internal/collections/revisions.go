package collections

import (
	"context"
	"errors"
	"strings"
	"time"
)

// MaxRevisionsPerRecord bounds the history kept for each record of a schema
// with Revisions: saving version n deletes revisions older than
// n-MaxRevisionsPerRecord+1.
const MaxRevisionsPerRecord = 50

var ErrRevisionNotFound = errors.New("revision not found")

// Revision is a snapshot of a record's data as saved at one version. The
// snapshot of the current version is kept too, so the history always ends
// with the live record.
type Revision struct {
	RecordID  string         `json:"record_id"`
	Version   int            `json:"version"`
	Data      map[string]any `json:"data"`
	Actor     string         `json:"actor,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// RevisionRepository is implemented by stores that keep revisions.
type RevisionRepository interface {
	// ListRevisions returns up to limit revisions of a record, newest first.
	ListRevisions(ctx context.Context, collectionID, recordID string, limit int) ([]Revision, error)
	GetRevision(ctx context.Context, collectionID, recordID string, version int) (Revision, error)
}

type actorKey struct{}

// WithActor records who makes the following writes; revisions store it.
func WithActor(ctx context.Context, actor string) context.Context {
	actor = strings.TrimSpace(actor)
	if len(actor) > 320 {
		actor = actor[:320]
	}
	return context.WithValue(ctx, actorKey{}, actor)
}

// ActorFromContext returns the actor set by WithActor, or "".
func ActorFromContext(ctx context.Context) string {
	actor, _ := ctx.Value(actorKey{}).(string)
	return actor
}

func (s Service) revisions(ctx context.Context, collectionID string) (RevisionRepository, error) {
	if s.Repository == nil {
		return nil, ErrCollectionNotFound
	}
	schema, err := s.Repository.GetSchema(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	repository, ok := s.Repository.(RevisionRepository)
	if !schema.Revisions || !ok {
		return nil, ErrRevisionNotFound
	}
	return repository, nil
}

// ListRevisions returns a record's newest revisions (at most limit, capped at
// MaxRevisionsPerRecord). The collection must have Revisions enabled.
func (s Service) ListRevisions(ctx context.Context, collectionID, recordID string, limit int) ([]Revision, error) {
	repository, err := s.revisions(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	if limit < 1 || limit > MaxRevisionsPerRecord {
		limit = MaxRevisionsPerRecord
	}
	return repository.ListRevisions(ctx, collectionID, recordID, limit)
}

// GetRevision returns one revision of a record.
func (s Service) GetRevision(ctx context.Context, collectionID, recordID string, version int) (Revision, error) {
	repository, err := s.revisions(ctx, collectionID)
	if err != nil {
		return Revision{}, err
	}
	if version < 1 {
		return Revision{}, ErrRevisionNotFound
	}
	return repository.GetRevision(ctx, collectionID, recordID, version)
}
