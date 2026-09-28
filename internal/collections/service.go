package collections

import (
	"context"
	"errors"
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
	DeleteSchema(context.Context, string) error
	ListRecords(context.Context, string, int, int) ([]Record, error)
	GetRecord(context.Context, string, string) (Record, error)
	SaveRecord(context.Context, string, string, map[string]any, []Field) (Record, error)
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
	if limit < 1 || limit > 100 || offset < 0 || offset > MaxCollectionRecords {
		return nil, ErrInvalidRecord
	}
	if _, err := s.GetSchema(ctx, collectionID); err != nil {
		return nil, err
	}
	return s.Repository.ListRecords(ctx, collectionID, limit, offset)
}

func (s Service) GetRecord(ctx context.Context, collectionID, id string) (Record, error) {
	if _, err := s.GetSchema(ctx, collectionID); err != nil {
		return Record{}, err
	}
	return s.Repository.GetRecord(ctx, collectionID, id)
}

func (s Service) CreateRecord(ctx context.Context, collectionID string, data map[string]any) (Record, error) {
	return s.saveRecord(ctx, collectionID, "", data)
}

func (s Service) UpdateRecord(ctx context.Context, collectionID, id string, data map[string]any) (Record, error) {
	if s.Repository == nil || id == "" {
		return Record{}, ErrRecordNotFound
	}
	if _, err := s.Repository.GetRecord(ctx, collectionID, id); err != nil {
		return Record{}, err
	}
	return s.saveRecord(ctx, collectionID, id, data)
}

func (s Service) saveRecord(ctx context.Context, collectionID, id string, data map[string]any) (Record, error) {
	if s.Repository == nil {
		return Record{}, ErrCollectionNotFound
	}
	schema, err := s.Repository.GetSchema(ctx, collectionID)
	if err != nil {
		return Record{}, err
	}
	clean, err := schema.ValidateRecord(data)
	if err != nil {
		return Record{}, err
	}
	unique := make([]Field, 0)
	for _, field := range schema.Fields {
		value, present := clean[field.ID]
		if !present {
			continue
		}
		if field.Unique {
			unique = append(unique, field)
		}
		if field.Kind != Relation && field.Kind != RelationMany {
			continue
		}
		relationIDs := []string{value.(string)}
		if field.Kind == RelationMany {
			items := value.([]any)
			relationIDs = relationIDs[:0]
			for _, item := range items {
				relationIDs = append(relationIDs, item.(string))
			}
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
	return s.Repository.SaveRecord(ctx, collectionID, id, clean, unique)
}

func (s Service) DeleteRecord(ctx context.Context, collectionID, id string) error {
	if s.Repository == nil {
		return ErrCollectionNotFound
	}
	if id == "" {
		return errors.New("record id is required")
	}
	return s.Repository.DeleteRecord(ctx, collectionID, id)
}
