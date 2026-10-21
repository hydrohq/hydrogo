package contenttypes

import (
	"errors"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// Revision callbacks for adminstudio.GeneratedAdminOptions. Collection
// storage writes a snapshot on every save of a type with `revisions: true`;
// these methods read and restore them.

var errNoRevisions = errors.New("content type has no revisions")

func (a AdminAPI) revisionType(id string) (Definition, error) {
	definition, ok := definitionByID(a.types(), id)
	if !ok || !definition.Revisions {
		return Definition{}, errNoRevisions
	}
	return definition, nil
}

// ListRevisions returns the newest revisions of a record.
func (a AdminAPI) ListRevisions(r *http.Request, collection adminstudio.Collection, id string, limit int) ([]adminstudio.RevisionSummary, error) {
	definition, err := a.revisionType(collection.ID)
	if err != nil {
		return nil, err
	}
	revisions, err := a.Service.ListRevisions(r.Context(), collection.ID, id, limit)
	if err != nil {
		return nil, err
	}
	result := make([]adminstudio.RevisionSummary, 0, len(revisions))
	for _, revision := range revisions {
		result = append(result, revisionSummary(definition, revision))
	}
	return result, nil
}

func revisionSummary(definition Definition, revision collections.Revision) adminstudio.RevisionSummary {
	summary := adminstudio.RevisionSummary{Version: int64(revision.Version), Actor: revision.Actor, CreatedAt: revision.CreatedAt}
	if definition.Drafts {
		summary.State = recordStatus(revision.Data)
	}
	return summary
}

// GetRevision returns a revision's fields formatted exactly like the current
// record in the admin, so the generated diff only shows real changes.
func (a AdminAPI) GetRevision(r *http.Request, collection adminstudio.Collection, id string, version int64) (adminstudio.RevisionSnapshot, error) {
	definition, err := a.revisionType(collection.ID)
	if err != nil {
		return adminstudio.RevisionSnapshot{}, err
	}
	revision, err := a.Service.GetRevision(r.Context(), collection.ID, id, int(version))
	if err != nil {
		return adminstudio.RevisionSnapshot{}, err
	}
	record := make(map[string]any, len(revision.Data)+1)
	for key, value := range revision.Data {
		record[key] = value
	}
	record = a.present(r, collection.ID, record)
	fields := make(map[string]string, len(collection.Fields))
	for _, field := range collection.Fields {
		if !field.ReadOnly {
			fields[field.ID] = adminstudio.DisplayValue(record[field.ID])
		}
	}
	return adminstudio.RevisionSnapshot{RevisionSummary: revisionSummary(definition, revision), Fields: fields}, nil
}

// CanRestore allows restoring any stored revision of a revisions type.
func (a AdminAPI) CanRestore(_ *http.Request, collection adminstudio.Collection, _ string, _ int64) bool {
	_, err := a.revisionType(collection.ID)
	return err == nil
}

// RestoreRevision saves a revision's data as a new version of the record,
// if the record is still at expectedVersion. Fields the type no longer has
// are dropped, and a drafts type keeps its current status: restoring content
// never publishes or unpublishes it.
func (a AdminAPI) RestoreRevision(r *http.Request, collection adminstudio.Collection, id string, version, expectedVersion int64) error {
	if _, err := a.revisionType(collection.ID); err != nil {
		return err
	}
	if expectedVersion < 1 {
		return collections.ErrVersionConflict
	}
	r = a.withActor(r)
	revision, err := a.Service.GetRevision(r.Context(), collection.ID, id, int(version))
	if err != nil {
		return err
	}
	schema, err := a.Service.GetSchema(r.Context(), collection.ID)
	if err != nil {
		return err
	}
	data := make(map[string]any, len(schema.Fields))
	for _, field := range schema.Fields {
		if value, ok := revision.Data[field.ID]; ok {
			data[field.ID] = value
		}
	}
	if err := a.keepStatus(r, collection.ID, id, data); err != nil {
		return err
	}
	if definition, ok := definitionByID(a.types(), collection.ID); ok {
		if err := a.keepHidden(r, definition, id, data); err != nil {
			return err
		}
		if err := a.runHooks(r, definition, HookEvent{On: "update", ID: id, Fields: data}); err != nil {
			return err
		}
	}
	_, err = a.Service.UpdateRecordVersion(r.Context(), collection.ID, id, data, int(expectedVersion))
	return err
}
