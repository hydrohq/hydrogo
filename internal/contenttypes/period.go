package contenttypes

import (
	"fmt"
	"net/http"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

// PeriodLock closes the books: records whose Field (a date) falls on or before
// the latest closing date can no longer be created, changed, moved on a state
// step or deleted, whatever their state. The closing dates live in records of
// the Closings type, in its date field Through; the latest one counts, so
// closing a period is adding a record, and who may do that is decided by that
// type's own permissions. Changing a record's date into a closed period is
// refused too.
//
//	period_lock: {field: issue_date, closings: period_closings, through: closed_through}
//
// Totals that the server rewrites on a parent because a line changed
// (rollups) are not checked against the period; lock the parent through a
// readonly rule as well.
type PeriodLock struct {
	Field    string `json:"field" yaml:"field"`
	Closings string `json:"closings" yaml:"closings"`
	Through  string `json:"through" yaml:"through"`
}

// CheckPeriodLocks verifies at startup that every lock names a date field of
// its own type and a date field of an existing closings type.
func CheckPeriodLocks(definitions []Definition) error {
	for _, definition := range definitions {
		lock := definition.PeriodLock
		if lock == nil {
			continue
		}
		own, found := fieldByID(definition, lock.Field)
		if !found || own.Kind != "date" {
			return fmt.Errorf("%w: period_lock.field of %q must be one of its date fields", ErrInvalidDefinition, definition.ID)
		}
		closings, ok := definitionByID(definitions, lock.Closings)
		if !ok || closings.ID == definition.ID {
			return fmt.Errorf("%w: period_lock.closings of %q must name another content type", ErrInvalidDefinition, definition.ID)
		}
		if through, found := fieldByID(closings, lock.Through); !found || through.Kind != "date" {
			return fmt.Errorf("%w: period_lock.through of %q must be a date field of %q", ErrInvalidDefinition, definition.ID, lock.Closings)
		}
	}
	return nil
}

// closedThrough returns the latest closing date ("" when no period is closed).
func (a AdminAPI) closedThrough(r *http.Request, lock PeriodLock) (string, error) {
	records, err := a.Service.QueryRecords(r.Context(), lock.Closings, collections.Query{Limit: 1, Sort: &collections.Sort{Field: lock.Through, Desc: true}})
	if err != nil || len(records) == 0 {
		return "", err
	}
	date, _ := records[0].Data[lock.Through].(string)
	return dateOnly(date), nil
}

func dateOnly(value string) string {
	if len(value) >= 10 {
		return value[:10]
	}
	return ""
}

// guardPeriod refuses a change that touches a closed period: the stored
// record (id non-empty) and the new values (next non-nil) must both be dated
// after the latest closing date.
func (a AdminAPI) guardPeriod(r *http.Request, definition Definition, id string, next map[string]any) error {
	lock := definition.PeriodLock
	if lock == nil {
		return nil
	}
	closed, err := a.closedThrough(r, *lock)
	if err != nil {
		return &hookError{http.StatusServiceUnavailable, "the closed periods could not be checked"}
	}
	if closed == "" {
		return nil
	}
	dates := []string{}
	if next != nil {
		if value, ok := next[lock.Field].(string); ok {
			dates = append(dates, dateOnly(value))
		}
	}
	if id != "" {
		stored, err := a.Service.GetRecord(r.Context(), definition.ID, id)
		if err != nil {
			return err
		}
		value, _ := stored.Data[lock.Field].(string)
		dates = append(dates, dateOnly(value))
	}
	for _, date := range dates {
		if date != "" && date <= closed {
			return &hookError{http.StatusUnprocessableEntity, "The period up to " + closed + " is closed"}
		}
	}
	return nil
}
