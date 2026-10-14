package collections

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
)

// SyncSchemas makes the stored schemas match the given definitions: missing
// schemas are created and changed ones replaced. Stored schemas that are not
// in the list are left alone. A change may add, remove or relabel fields and
// change options or constraints, but an existing field may not change its
// kind or relation target, because stored records would no longer validate.
func (s Service) SyncSchemas(ctx context.Context, schemas []Schema) error {
	ordered, err := orderByRelations(schemas)
	if err != nil {
		return err
	}
	for _, schema := range ordered {
		existing, err := s.GetSchema(ctx, schema.ID)
		switch {
		case errors.Is(err, ErrCollectionNotFound):
			if err := s.CreateSchema(ctx, schema); err != nil {
				return fmt.Errorf("create %s: %w", schema.ID, err)
			}
		case err != nil:
			return err
		case reflect.DeepEqual(existing, schema):
		default:
			if err := s.ReplaceSchema(ctx, existing, schema); err != nil {
				return fmt.Errorf("update %s: %w", schema.ID, err)
			}
		}
	}
	return nil
}

// ReplaceSchema stores next in place of current after checking that records
// written under current stay valid.
func (s Service) ReplaceSchema(ctx context.Context, current, next Schema) error {
	if s.Repository == nil {
		return ErrCollectionNotFound
	}
	if err := next.Validate(); err != nil {
		return err
	}
	if current.ID != next.ID {
		return ErrInvalidSchema
	}
	before := make(map[string]Field, len(current.Fields))
	for _, field := range current.Fields {
		before[field.ID] = field
	}
	for _, field := range next.Fields {
		if old, ok := before[field.ID]; ok && (old.Kind != field.Kind || old.Relation != field.Relation) {
			return fmt.Errorf("%w: field %q cannot change from %s to %s; add a new field instead", ErrInvalidSchema, field.ID, old.Kind, field.Kind)
		}
	}
	return s.Repository.UpdateSchema(ctx, next)
}

// orderByRelations sorts schemas so that relation targets come first.
// Self-references are allowed; cycles between schemas are not.
func orderByRelations(schemas []Schema) ([]Schema, error) {
	byID := make(map[string]Schema, len(schemas))
	ids := make([]string, 0, len(schemas))
	for _, schema := range schemas {
		if _, dup := byID[schema.ID]; dup {
			return nil, fmt.Errorf("%w: duplicate schema %q", ErrInvalidSchema, schema.ID)
		}
		byID[schema.ID] = schema
		ids = append(ids, schema.ID)
	}
	sort.Strings(ids)
	const (
		visiting = 1
		done     = 2
	)
	state := make(map[string]int, len(ids))
	ordered := make([]Schema, 0, len(ids))
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case done:
			return nil
		case visiting:
			return fmt.Errorf("%w: relation cycle through %q", ErrInvalidSchema, id)
		}
		state[id] = visiting
		for _, field := range byID[id].Fields {
			if (field.Kind == Relation || field.Kind == RelationMany) && field.Relation != id {
				if _, ok := byID[field.Relation]; ok {
					if err := visit(field.Relation); err != nil {
						return err
					}
				}
			}
		}
		state[id] = done
		ordered = append(ordered, byID[id])
		return nil
	}
	for _, id := range ids {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}
