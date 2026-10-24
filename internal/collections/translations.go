package collections

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"time"
)

// Localization: a record holds the default locale's values of every field.
// Values of localized fields (Field.Localized) in other locales are stored as
// translations, one per record and locale, each with its own version for
// optimistic locking. Translations are deleted with their record. Status,
// revisions, search and uniqueness only ever look at the record itself.

// MaxTranslationBatch bounds the record IDs of one ListTranslations call.
const MaxTranslationBatch = 100

var (
	ErrNotLocalized  = errors.New("collection has no localized fields")
	ErrInvalidLocale = errors.New("invalid locale")
)

var localePattern = regexp.MustCompile(`^[a-z]{2,8}(-[a-z0-9]{1,8}){0,3}$`)

// ValidLocale reports whether locale is a lowercase BCP 47-like tag
// ("en", "pt-br") that can be stored.
func ValidLocale(locale string) bool {
	return len(locale) <= 35 && localePattern.MatchString(locale)
}

// Translation holds one record's localized values in one locale. Version is
// 0 when nothing has been saved for that locale yet.
type Translation struct {
	RecordID  string         `json:"record_id"`
	Locale    string         `json:"locale"`
	Data      map[string]any `json:"data"`
	Version   int            `json:"version"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// TranslationRepository is implemented by stores that keep translations.
type TranslationRepository interface {
	// GetTranslation fails with ErrRecordNotFound for a missing record and
	// returns a zero-version translation when the locale has no values yet.
	GetTranslation(ctx context.Context, collectionID, recordID, locale string) (Translation, error)
	// ListTranslations returns the stored translations of the given records;
	// records without one are absent from the map.
	ListTranslations(ctx context.Context, collectionID string, recordIDs []string, locale string) (map[string]Translation, error)
	// SaveTranslation inserts (expectedVersion 0) or updates (stored version
	// equals expectedVersion) a translation, else ErrVersionConflict.
	SaveTranslation(ctx context.Context, collectionID, recordID, locale string, data map[string]any, expectedVersion int) (Translation, error)
}

func localizable(kind FieldKind) bool {
	return kind == Text || kind == RichText || kind == Media || kind == Select
}

// Localized reports whether the schema has localized fields.
func (s Schema) Localized() bool {
	for _, field := range s.Fields {
		if field.Localized {
			return true
		}
	}
	return false
}

// ValidateTranslation checks translated values: only localized fields, each
// valid for its kind. Empty values are dropped (the locale then falls back
// to the record's value); required is not enforced because the record
// already holds a value for every required field.
func (s Schema) ValidateTranslation(data map[string]any) (map[string]any, error) {
	if err := s.Validate(); err != nil || len(data) > MaxFields {
		return nil, ErrInvalidRecord
	}
	fields := make(map[string]Field, len(s.Fields))
	for _, field := range s.Fields {
		if field.Localized {
			fields[field.ID] = field
		}
	}
	clean := make(map[string]any, len(data))
	for id, value := range data {
		field, ok := fields[id]
		if !ok {
			return nil, fieldError(id)
		}
		if value == nil || value == "" {
			continue
		}
		if err := validateValue(field, value); err != nil {
			return nil, fieldError(id)
		}
		clean[id] = storedValue(field, value)
	}
	encoded, err := json.Marshal(clean)
	if err != nil || len(encoded) > MaxRecordBytes {
		return nil, ErrInvalidRecord
	}
	return clean, nil
}

// onlyLocalized drops stored values of fields that are no longer localized.
func (s Schema) onlyLocalized(data map[string]any) map[string]any {
	result := make(map[string]any, len(data))
	for _, field := range s.Fields {
		if value, ok := data[field.ID]; ok && field.Localized {
			result[field.ID] = value
		}
	}
	return result
}

func (s Service) translations(ctx context.Context, collectionID, locale string) (Schema, TranslationRepository, error) {
	if s.Repository == nil {
		return Schema{}, nil, ErrCollectionNotFound
	}
	schema, err := s.Repository.GetSchema(ctx, collectionID)
	if err != nil {
		return Schema{}, nil, err
	}
	repository, ok := s.Repository.(TranslationRepository)
	if !schema.Localized() || !ok {
		return Schema{}, nil, ErrNotLocalized
	}
	if !ValidLocale(locale) {
		return Schema{}, nil, ErrInvalidLocale
	}
	return schema, repository, nil
}

// GetTranslation returns a record's values in locale (version 0 and no data
// when none were saved).
func (s Service) GetTranslation(ctx context.Context, collectionID, recordID, locale string) (Translation, error) {
	schema, repository, err := s.translations(ctx, collectionID, locale)
	if err != nil {
		return Translation{}, err
	}
	translation, err := repository.GetTranslation(ctx, collectionID, recordID, locale)
	if err != nil {
		return Translation{}, err
	}
	translation.Data = schema.onlyLocalized(translation.Data)
	return translation, nil
}

// ListTranslations returns the stored translations of up to
// MaxTranslationBatch records, keyed by record ID.
func (s Service) ListTranslations(ctx context.Context, collectionID string, recordIDs []string, locale string) (map[string]Translation, error) {
	schema, repository, err := s.translations(ctx, collectionID, locale)
	if err != nil {
		return nil, err
	}
	if len(recordIDs) > MaxTranslationBatch {
		return nil, ErrInvalidQuery
	}
	if len(recordIDs) == 0 {
		return map[string]Translation{}, nil
	}
	result, err := repository.ListTranslations(ctx, collectionID, recordIDs, locale)
	if err != nil {
		return nil, err
	}
	for id, translation := range result {
		translation.Data = schema.onlyLocalized(translation.Data)
		result[id] = translation
	}
	return result, nil
}

// SaveTranslation replaces a record's values in locale. expectedVersion is
// the translation's version as loaded (0 for a locale never saved); a
// mismatch fails with ErrVersionConflict. Data is checked with
// ValidateTranslation. The default locale is not stored here: its values
// are the record's own, saved with UpdateRecordVersion.
func (s Service) SaveTranslation(ctx context.Context, collectionID, recordID, locale string, data map[string]any, expectedVersion int) (Translation, error) {
	schema, repository, err := s.translations(ctx, collectionID, locale)
	if err != nil {
		return Translation{}, err
	}
	if expectedVersion < 0 {
		return Translation{}, ErrVersionConflict
	}
	clean, err := schema.ValidateTranslation(data)
	if err != nil {
		return Translation{}, err
	}
	return repository.SaveTranslation(ctx, collectionID, recordID, locale, clean, expectedVersion)
}
