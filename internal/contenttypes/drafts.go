package contenttypes

import (
	"errors"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// Types with `drafts: true` store a status field in each record. The admin
// changes it only through the workflow actions below (wired as
// GeneratedAdminOptions.CanTransition/Transition); record saves keep the
// stored status, and the public API filters on it.
const (
	StatusField     = "status"
	StatusDraft     = "draft"
	StatusPublished = "published"
)

var errTransition = errors.New("content type transition is not allowed")

// recordStatus returns a record's status; records without one (written
// before drafts were enabled) are drafts.
func recordStatus(data map[string]any) string {
	if status, _ := data[StatusField].(string); status == StatusPublished {
		return StatusPublished
	}
	return StatusDraft
}

// publishedCondition is the filter the public API adds for drafts types.
func publishedCondition() collections.Condition {
	return collections.Condition{Field: StatusField, Op: collections.Equals, Value: StatusPublished}
}

func definitionByID(types []Definition, id string) (Definition, bool) {
	for _, definition := range types {
		if definition.ID == id {
			return definition, true
		}
	}
	return Definition{}, false
}

// CanTransition allows draft <-> published for drafts types. Who may use the
// admin at all is decided by GeneratedAdminOptions.Authorize.
func (a AdminAPI) CanTransition(_ *http.Request, collection adminstudio.Collection, _ map[string]any, target adminstudio.PublicationState) bool {
	definition, ok := definitionByID(a.Types, collection.ID)
	return ok && definition.Drafts && (target == adminstudio.StateDraft || target == adminstudio.StatePublished)
}

// Transition sets a record's status, failing with a version conflict when
// the record changed since expectedVersion.
func (a AdminAPI) Transition(r *http.Request, collection adminstudio.Collection, record map[string]any, target adminstudio.PublicationState, expectedVersion int64) error {
	if !a.CanTransition(r, collection, record, target) || expectedVersion < 1 {
		return errTransition
	}
	id, _ := record["id"].(string)
	stored, err := a.Service.GetRecord(r.Context(), collection.ID, id)
	if err != nil {
		return err
	}
	data := make(map[string]any, len(stored.Data)+1)
	for key, value := range stored.Data {
		data[key] = value
	}
	data[StatusField] = string(target)
	_, err = a.Service.UpdateRecordVersion(r.Context(), collection.ID, id, data, int(expectedVersion))
	return err
}

// keepStatus makes the stored status authoritative on save: new records are
// drafts and updates keep the current status, whatever the payload says.
func (a AdminAPI) keepStatus(r *http.Request, typeID, id string, data map[string]any) error {
	definition, ok := definitionByID(a.Types, typeID)
	if !ok || !definition.Drafts {
		return nil
	}
	if id == "" {
		data[StatusField] = StatusDraft
		return nil
	}
	stored, err := a.Service.GetRecord(r.Context(), typeID, id)
	if err != nil {
		return err
	}
	data[StatusField] = recordStatus(stored.Data)
	return nil
}
