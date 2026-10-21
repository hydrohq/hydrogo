package collections

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Record struct {
	ID           string         `json:"id"`
	CollectionID string         `json:"collection_id"`
	Data         map[string]any `json:"data"`
	Version      int            `json:"version"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type Repository interface {
	ListSchemas(context.Context) ([]Schema, error)
	GetSchema(context.Context, string) (Schema, error)
	CreateSchema(context.Context, Schema) error
	UpdateSchema(context.Context, Schema) error
	DeleteSchema(context.Context, string) error
	// QueryRecords lists records matching a query already checked by
	// Schema.ValidateQuery.
	QueryRecords(ctx context.Context, collectionID string, query Query) ([]Record, error)
	GetRecord(context.Context, string, string) (Record, error)
	// SaveRecord creates (empty id) or updates a record. When expectedVersion
	// is positive the update only applies if the stored version still matches,
	// otherwise it fails with ErrVersionConflict.
	SaveRecord(ctx context.Context, collectionID, id string, data map[string]any, unique []Field, expectedVersion int) (Record, error)
	DeleteRecord(context.Context, string, string) error
	RecordExists(context.Context, string, string) (bool, error)
}

type Service struct{ Repository Repository }

func (s Service) ListSchemas(ctx context.Context) ([]Schema, error) {
	if s.Repository == nil {
		return nil, ErrCollectionNotFound
	}
	return s.Repository.ListSchemas(ctx)
}

func (s Service) GetSchema(ctx context.Context, id string) (Schema, error) {
	if s.Repository == nil {
		return Schema{}, ErrCollectionNotFound
	}
	return s.Repository.GetSchema(ctx, id)
}

func (s Service) CreateSchema(ctx context.Context, schema Schema) error {
	if s.Repository == nil {
		return ErrCollectionNotFound
	}
	if err := schema.Validate(); err != nil {
		return err
	}
	existing, err := s.Repository.ListSchemas(ctx)
	if err != nil {
		return err
	}
	if len(existing) >= MaxCollections {
		return ErrInvalidSchema
	}
	for _, other := range existing {
		if other.ID == schema.ID {
			return ErrInvalidSchema
		}
	}
	available := make(map[string]struct{}, len(existing))
	for _, other := range existing {
		available[other.ID] = struct{}{}
	}
	for _, field := range schema.Fields {
		if field.Kind == Relation || field.Kind == RelationMany {
			if field.Relation == schema.ID {
				continue
			}
			if _, ok := available[field.Relation]; !ok {
				return ErrInvalidSchema
			}
		}
	}
	return s.Repository.CreateSchema(ctx, schema)
}

func (s Service) DeleteSchema(ctx context.Context, id string) error {
	if s.Repository == nil {
		return ErrCollectionNotFound
	}
	if _, err := s.Repository.GetSchema(ctx, id); err != nil {
		return err
	}
	schemas, err := s.Repository.ListSchemas(ctx)
	if err != nil {
		return err
	}
	for _, schema := range schemas {
		if schema.ID == id {
			continue
		}
		for _, field := range schema.Fields {
			if (field.Kind == Relation || field.Kind == RelationMany) && field.Relation == id {
				return ErrInvalidSchema
			}
		}
	}
	return s.Repository.DeleteSchema(ctx, id)
}

func (s Service) ListRecords(ctx context.Context, collectionID string, limit, offset int) ([]Record, error) {
	if limit < 1 || limit > 100 || offset < 0 || offset > MaxListOffset {
		return nil, ErrInvalidRecord
	}
	return s.QueryRecords(ctx, collectionID, Query{Limit: limit, Offset: offset})
}

// QueryRecords validates the query against the collection schema and returns
// the matching page of records. Without a sort, records come newest first.
func (s Service) QueryRecords(ctx context.Context, collectionID string, query Query) ([]Record, error) {
	schema, err := s.GetSchema(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	checked, err := schema.ValidateQuery(query)
	if err != nil {
		return nil, err
	}
	return s.Repository.QueryRecords(ctx, collectionID, checked)
}

func (s Service) GetRecord(ctx context.Context, collectionID, id string) (Record, error) {
	if _, err := s.GetSchema(ctx, collectionID); err != nil {
		return Record{}, err
	}
	return s.Repository.GetRecord(ctx, collectionID, id)
}

func (s Service) CreateRecord(ctx context.Context, collectionID string, data map[string]any) (Record, error) {
	return s.saveRecord(ctx, collectionID, "", data, 0)
}

func (s Service) UpdateRecord(ctx context.Context, collectionID, id string, data map[string]any) (Record, error) {
	return s.UpdateRecordVersion(ctx, collectionID, id, data, 0)
}

// UpdateRecordVersion updates a record only if its stored version equals
// expectedVersion (optimistic locking). An expectedVersion of 0 skips the check.
func (s Service) UpdateRecordVersion(ctx context.Context, collectionID, id string, data map[string]any, expectedVersion int) (Record, error) {
	if s.Repository == nil || id == "" {
		return Record{}, ErrRecordNotFound
	}
	if _, err := s.Repository.GetRecord(ctx, collectionID, id); err != nil {
		return Record{}, err
	}
	return s.saveRecord(ctx, collectionID, id, data, expectedVersion)
}

func (s Service) saveRecord(ctx context.Context, collectionID, id string, data map[string]any, expectedVersion int) (Record, error) {
	if s.Repository == nil {
		return Record{}, ErrCollectionNotFound
	}
	schema, err := s.Repository.GetSchema(ctx, collectionID)
	if err != nil {
		return Record{}, err
	}
	if len(schema.RollupParents) == 0 {
		return s.saveChecked(ctx, schema, id, data, expectedVersion)
	}
	return s.withRollups(ctx, schema, id, func(tx Service) (Record, error) {
		return tx.saveChecked(ctx, schema, id, data, expectedVersion)
	})
}

// saveChecked validates and saves one record of schema.
func (s Service) saveChecked(ctx context.Context, schema Schema, id string, data map[string]any, expectedVersion int) (Record, error) {
	collectionID := schema.ID
	clean, err := schema.ValidateRecord(data)
	if err != nil {
		return Record{}, err
	}
	var stored *Record
	if id != "" && (len(schema.sequenceFields()) > 0 || schema.States != nil || schema.hasReadonlyRules() || len(schema.rollupFields()) > 0) {
		record, err := s.Repository.GetRecord(ctx, collectionID, id)
		if err != nil {
			return Record{}, err
		}
		stored = &record
		if err := schema.checkReadonly(record.Data); err != nil {
			return Record{}, err
		}
	}
	if machine := schema.States; machine != nil {
		// The state is never taken from the client: new records start in
		// the initial state, updates keep the stored one (see Transition).
		clean[machine.Field] = machine.Initial
		if stored != nil {
			if value, ok := stored.Data[machine.Field]; ok {
				clean[machine.Field] = value
			}
		}
	}
	return s.persist(ctx, schema, id, clean, stored, expectedVersion)
}

// persist runs the asserts and writes a validated record: sequence numbers
// (kept on update), unique values, related records and the save itself.
func (s Service) persist(ctx context.Context, schema Schema, id string, clean map[string]any, stored *Record, expectedVersion int) (Record, error) {
	collectionID := schema.ID
	sequences := schema.sequenceFields()
	if stored != nil {
		// Numbers never change: keep the stored ones.
		for _, field := range sequences {
			if value, ok := stored.Data[field.ID]; ok {
				clean[field.ID] = value
			}
		}
	}
	if err := schema.carryRollups(clean, stored); err != nil {
		return Record{}, err
	}
	if err := schema.checkAsserts(clean); err != nil {
		return Record{}, err
	}
	unique := make([]Field, 0)
	for _, field := range schema.Fields {
		value, present := clean[field.ID]
		if !present {
			continue
		}
		if field.Unique || field.Sequence != nil {
			unique = append(unique, field)
		}
		if field.Kind != Relation && field.Kind != RelationMany {
			continue
		}
		var relationIDs []string
		if field.Kind == RelationMany {
			for _, item := range value.([]any) {
				relationIDs = append(relationIDs, item.(string))
			}
		} else {
			relationIDs = []string{value.(string)}
		}
		for _, relatedID := range relationIDs {
			exists, err := s.Repository.RecordExists(ctx, field.Relation, relatedID)
			if err != nil {
				return Record{}, err
			}
			if !exists {
				return Record{}, fieldError(field.ID)
			}
		}
	}
	if len(sequences) > 0 && id == "" {
		return s.createNumbered(ctx, collectionID, clean, unique, sequences)
	}
	return s.Repository.SaveRecord(ctx, collectionID, id, clean, unique, expectedVersion)
}

// Transition moves a record to another state of the type's state machine. It
// fails with ErrTransitionNotAllowed when the machine has no such step from
// the current state or the step's condition does not hold, and with
// ErrTransitionForbidden when the step needs a permission can does not grant.
// A readonly rule does not block a transition, so a locked record can still be
// moved on (for example voided); the asserts apply to the record as it will be.
func (s Service) Transition(ctx context.Context, collectionID, id, to string, expectedVersion int, can func(permission string) bool) (Record, error) {
	if s.Repository == nil {
		return Record{}, ErrCollectionNotFound
	}
	schema, err := s.Repository.GetSchema(ctx, collectionID)
	if err != nil {
		return Record{}, err
	}
	machine := schema.States
	if machine == nil {
		return Record{}, ErrNoStates
	}
	stored, err := s.Repository.GetRecord(ctx, collectionID, id)
	if err != nil {
		return Record{}, err
	}
	from, _ := stored.Data[machine.Field].(string)
	if from == "" {
		from = machine.Initial
	}
	var step *Transition
	for i := range machine.Transitions {
		if machine.Transitions[i].From == from && machine.Transitions[i].To == to {
			step = &machine.Transitions[i]
		}
	}
	if step == nil {
		return Record{}, fmt.Errorf("%w: %s to %s", ErrTransitionNotAllowed, from, to)
	}
	if step.Permission != "" && (can == nil || !can(step.Permission)) {
		return Record{}, ErrTransitionForbidden
	}
	if step.When != "" {
		ok, err := holds(step.When, stored.Data)
		if err != nil || !ok {
			message := strings.TrimSpace(step.Message)
			if message == "" {
				message = from + " to " + to + " is not possible yet"
			}
			return Record{}, fmt.Errorf("%w: %s", ErrTransitionNotAllowed, message)
		}
	}
	data := make(map[string]any, len(stored.Data))
	for key, value := range stored.Data {
		data[key] = value
	}
	clean, err := schema.ValidateRecord(data)
	if err != nil {
		return Record{}, err
	}
	clean[machine.Field] = to
	return s.persist(ctx, schema, id, clean, &stored, expectedVersion)
}

// createNumbered takes the next number of every sequence field and saves the
// record in one transaction, so a failed save leaves no gap.
func (s Service) createNumbered(ctx context.Context, collectionID string, clean map[string]any, unique []Field, sequences []Field) (Record, error) {
	var record Record
	err := s.InTransaction(ctx, func(tx Service) error {
		numbers, ok := tx.Repository.(SequenceRepository)
		if !ok {
			return ErrNoTransactions
		}
		now := time.Now()
		for _, field := range sequences {
			scope := field.Sequence.Scope(now)
			n, err := numbers.NextSequence(ctx, collectionID, field.ID, scope)
			if err != nil {
				return err
			}
			clean[field.ID] = field.Sequence.Format(scope, n)
		}
		var err error
		record, err = tx.Repository.SaveRecord(ctx, collectionID, "", clean, append(unique, sequences...), 0)
		return err
	})
	return record, err
}

func (s Service) DeleteRecord(ctx context.Context, collectionID, id string) error {
	if s.Repository == nil {
		return ErrCollectionNotFound
	}
	if id == "" {
		return errors.New("record id is required")
	}
	if schema, err := s.Repository.GetSchema(ctx, collectionID); err == nil && len(schema.RollupParents) > 0 {
		_, err := s.withRollups(ctx, schema, id, func(tx Service) (Record, error) {
			return Record{}, tx.deleteChecked(ctx, collectionID, id)
		})
		return err
	}
	return s.deleteChecked(ctx, collectionID, id)
}

func (s Service) deleteChecked(ctx context.Context, collectionID, id string) error {
	if schema, err := s.Repository.GetSchema(ctx, collectionID); err == nil && schema.hasReadonlyRules() {
		stored, err := s.Repository.GetRecord(ctx, collectionID, id)
		if err != nil {
			return err
		}
		if err := schema.checkReadonly(stored.Data); err != nil {
			return err
		}
	}
	return s.Repository.DeleteRecord(ctx, collectionID, id)
}
