// Package contenttypes turns YAML content-type definitions into the two
// runtime shapes HydroGo needs: a collections.Schema for storage and
// validation, and an adminstudio.Collection for the generic admin screens.
//
// One file per type lives in the content-types directory:
//
//	id: faq
//	label: FAQ
//	title_field: question
//	public: true
//	drafts: true
//	revisions: true
//	fields:
//	  - {id: question, label: Question, kind: text, required: true, list: true, searchable: true}
//	  - {id: answer, label: Answer, kind: richtext, required: true, searchable: true}
//	  - {id: category, label: Category, kind: select, options: [general, billing], list: true, filterable: true}
//	  - {id: owner, label: Owner, kind: relation, to: team}
//
// Type keys: id, label, description, group, icon, title_field, fields, and:
//
//   - public: serve published records at /api/v1/content/{id}.
//   - drafts: give records a draft/published status (a "status" field the
//     admin changes with Publish and Move to draft). New records are drafts;
//     the public API never returns drafts.
//   - revisions: keep the last 50 saved versions of each record. The edit
//     page lists them; each can be compared with the current record and
//     restored (restoring saves a new version and keeps the current status).
//
// Field keys: id, label, kind, required, unique, options (select), to and
// many (relation), min and max (number), and:
//
//   - list: show the field as a column in the admin list. List columns are
//     sortable by clicking their heading.
//   - searchable: include the field in the admin list search and the public
//     API's ?search= (text, textarea, richtext, select).
//   - filterable: add a filter for the field to the admin list and allow
//     ?filter[field]= on the public API (text, textarea, select, number,
//     boolean, date, datetime and single relation fields). Beyond equality
//     the API accepts filter[field][ne|gt|gte|lt|lte|in|null]=value; gt, gte,
//     lt and lte need a number or date field, in takes a comma separated
//     list (at most 50 values) and null takes true or false.
//   - localized: keep one value per site locale (text, textarea, richtext,
//     media; not together with unique, searchable or filterable). A type
//     with localized fields needs revisions: true. The record holds the
//     default locale's value; the admin edits the other locales through
//     its locale tabs, and the public API's ?locale= returns them, falling
//     back to the default locale per field. Localization is active only
//     when the site has i18n enabled with more than one locale; otherwise
//     the flag changes nothing.
//
// The public API also accepts ?sort=field or ?sort=-field on any field and
// on created_at and updated_at.
package contenttypes

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

var ErrInvalidDefinition = errors.New("invalid content type definition")

// reserved field IDs are added by the storage layer to every record.
// "locale" is added to public API responses.
var reserved = map[string]bool{"id": true, "version": true, "created_at": true, "updated_at": true, "locale": true}

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// Definition is one content type as written in YAML.
type Definition struct {
	ID          string `json:"id" yaml:"id"`
	Label       string `json:"label" yaml:"label"`
	Description string `json:"description,omitempty" yaml:"description"`
	Group       string `json:"group,omitempty" yaml:"group"`
	Icon        string `json:"icon,omitempty" yaml:"icon"`
	TitleField  string `json:"title_field,omitempty" yaml:"title_field"`
	Public      bool   `json:"public" yaml:"public"`
	// Drafts adds a draft/published status to every record. New records
	// start as drafts, the admin publishes and unpublishes them, and the
	// public API only returns published records.
	Drafts bool `json:"drafts" yaml:"drafts"`
	// Revisions keeps the last collections.MaxRevisionsPerRecord saved
	// versions of each record; the admin can view and restore them.
	Revisions bool    `json:"revisions" yaml:"revisions"`
	Fields    []Field `json:"fields" yaml:"fields"`
}

// Field kinds: text, textarea, richtext, number, boolean, date, datetime,
// select, media, relation (with to and optionally many).
type Field struct {
	ID       string   `json:"id" yaml:"id"`
	Label    string   `json:"label" yaml:"label"`
	Kind     string   `json:"kind" yaml:"kind"`
	Required bool     `json:"required" yaml:"required"`
	Unique   bool     `json:"unique" yaml:"unique"`
	List     bool     `json:"list" yaml:"list"`
	Options  []string `json:"options,omitempty" yaml:"options"`
	To       string   `json:"to,omitempty" yaml:"to"`
	Many     bool     `json:"many,omitempty" yaml:"many"`
	Min      *float64 `json:"min,omitempty" yaml:"min"`
	Max      *float64 `json:"max,omitempty" yaml:"max"`
	// Searchable fields (text, textarea, richtext, select) are matched by
	// the admin list search box and the public API's search parameter.
	Searchable bool `json:"searchable" yaml:"searchable"`
	// Filterable fields (see the package documentation) get a filter in
	// the admin list and accept filter[field]= on the public API.
	Filterable bool `json:"filterable" yaml:"filterable"`
	// Localized fields (text, textarea, richtext, media) have a value per
	// site locale; see the package documentation.
	Localized bool `json:"localized" yaml:"localized"`
}

// LoadDir reads every *.yaml file in dir. A missing directory yields no
// definitions. Unknown YAML keys are rejected so typos fail loudly.
func LoadDir(dir string) ([]Definition, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	definitions := make([]Definition, 0, len(paths))
	seen := map[string]string{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var definition Definition
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&definition); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if err := definition.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if other, dup := seen[definition.ID]; dup {
			return nil, fmt.Errorf("%s: %w: id %q is already defined in %s", path, ErrInvalidDefinition, definition.ID, other)
		}
		seen[definition.ID] = path
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

// Validate checks the definition and that both derived shapes are valid.
func (d Definition) Validate() error {
	if !identifier.MatchString(d.ID) {
		return fmt.Errorf("%w: id %q must be lowercase letters, digits and underscores", ErrInvalidDefinition, d.ID)
	}
	if strings.TrimSpace(d.Label) == "" {
		return fmt.Errorf("%w: label is required", ErrInvalidDefinition)
	}
	for _, field := range d.Fields {
		if reserved[field.ID] {
			return fmt.Errorf("%w: field id %q is reserved", ErrInvalidDefinition, field.ID)
		}
		if d.Drafts && field.ID == StatusField {
			return fmt.Errorf("%w: field id %q is reserved for drafts", ErrInvalidDefinition, field.ID)
		}
		if field.Kind == "relation" && strings.TrimSpace(field.To) == "" {
			return fmt.Errorf("%w: relation field %q needs to:", ErrInvalidDefinition, field.ID)
		}
		if field.Kind != "relation" && (field.To != "" || field.Many) {
			return fmt.Errorf("%w: only relation fields take to/many (field %q)", ErrInvalidDefinition, field.ID)
		}
		if field.Searchable && !oneOf(field.Kind, "text", "textarea", "richtext", "select") {
			return fmt.Errorf("%w: field %q of kind %s cannot be searchable", ErrInvalidDefinition, field.ID, field.Kind)
		}
		if field.Filterable && (!oneOf(field.Kind, "text", "textarea", "select", "number", "boolean", "date", "datetime", "relation") || field.Many) {
			return fmt.Errorf("%w: field %q of kind %s cannot be filterable", ErrInvalidDefinition, field.ID, field.Kind)
		}
		if field.Localized && !oneOf(field.Kind, "text", "textarea", "richtext", "media") {
			return fmt.Errorf("%w: field %q of kind %s cannot be localized", ErrInvalidDefinition, field.ID, field.Kind)
		}
		if field.Localized && (field.Unique || field.Searchable || field.Filterable) {
			return fmt.Errorf("%w: localized field %q cannot be unique, searchable or filterable", ErrInvalidDefinition, field.ID)
		}
	}
	if d.Localized() && !d.Revisions {
		return fmt.Errorf("%w: a type with localized fields needs revisions: true", ErrInvalidDefinition)
	}
	schema, err := d.Schema()
	if err != nil {
		return err
	}
	if err := schema.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDefinition, err)
	}
	if _, err := adminstudio.NewRegistry([]adminstudio.Collection{d.AdminCollection()}); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDefinition, err)
	}
	return nil
}

// Schema returns the storage schema.
func (d Definition) Schema() (collections.Schema, error) {
	schema := collections.Schema{ID: d.ID, Label: d.Label, Description: d.Description, Revisions: d.Revisions}
	for _, field := range d.Fields {
		kind, err := storageKind(field)
		if err != nil {
			return collections.Schema{}, err
		}
		schema.Fields = append(schema.Fields, collections.Field{
			ID: field.ID, Label: field.Label, Kind: kind, Required: field.Required, Unique: field.Unique,
			Min: field.Min, Max: field.Max, Options: append([]string(nil), field.Options...), Relation: field.To,
			Localized: field.Localized, Indexed: field.Filterable,
		})
	}
	if d.Drafts {
		// Not required: records written before drafts were enabled have no
		// status and count as drafts (see recordStatus).
		schema.Fields = append(schema.Fields, collections.Field{ID: StatusField, Label: "Status", Kind: collections.Select, Options: []string{StatusDraft, StatusPublished}, Indexed: true})
	}
	return schema, nil
}

// Localized reports whether any field is localized.
func (d Definition) Localized() bool {
	for _, field := range d.Fields {
		if field.Localized {
			return true
		}
	}
	return false
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func storageKind(field Field) (collections.FieldKind, error) {
	switch field.Kind {
	case "text", "textarea":
		return collections.Text, nil
	case "richtext":
		return collections.RichText, nil
	case "number":
		return collections.Number, nil
	case "boolean":
		return collections.Boolean, nil
	case "date", "datetime":
		return collections.DateTime, nil
	case "select":
		return collections.Select, nil
	case "media":
		return collections.Media, nil
	case "relation":
		if field.Many {
			return collections.RelationMany, nil
		}
		return collections.Relation, nil
	default:
		return "", fmt.Errorf("%w: field %q has unknown kind %q", ErrInvalidDefinition, field.ID, field.Kind)
	}
}

// AdminCollection returns the metadata the admin renders its generic
// screens from.
func (d Definition) AdminCollection() adminstudio.Collection {
	collection := adminstudio.Collection{
		ID: d.ID, Label: d.Label, Description: d.Description, Group: d.Group, Icon: d.Icon,
		TitleField: d.TitleField, Versioned: true, Revisions: d.Revisions, Localization: d.Localized(),
		Source: adminstudio.SourceContentType, APIPath: AdminAPIPath + "/" + d.ID,
	}
	if collection.Group == "" {
		collection.Group = "Content"
	}
	for _, field := range d.Fields {
		admin := adminstudio.Field{
			ID: field.ID, Label: field.Label, Required: field.Required, VisibleInList: field.List,
			Searchable: field.Searchable, Filterable: field.Filterable, Options: append([]string(nil), field.Options...),
			Localized: field.Localized,
		}
		switch field.Kind {
		case "text":
			admin.Kind = adminstudio.FieldText
		case "textarea":
			admin.Kind, admin.Widget = adminstudio.FieldText, "textarea"
		case "richtext":
			admin.Kind = adminstudio.FieldRichText
		case "number":
			admin.Kind = adminstudio.FieldNumber
		case "boolean":
			admin.Kind = adminstudio.FieldBoolean
		case "date":
			admin.Kind = adminstudio.FieldDate
		case "datetime":
			admin.Kind, admin.Widget = adminstudio.FieldDate, "datetime-local"
		case "select":
			admin.Kind = adminstudio.FieldSelect
		case "media":
			admin.Kind = adminstudio.FieldMedia
		case "relation":
			admin.Kind = adminstudio.FieldRelation
			cardinality := "one"
			if field.Many {
				cardinality = "many"
			}
			admin.Relation = &adminstudio.RelationRef{Collection: field.To, Cardinality: cardinality}
		default:
			admin.Kind = adminstudio.FieldKind(field.Kind)
		}
		collection.Fields = append(collection.Fields, admin)
	}
	if d.Drafts {
		collection.Fields = append(collection.Fields, adminstudio.Field{
			ID: StatusField, Label: "Status", Kind: adminstudio.FieldSelect, VisibleInList: true, Filterable: true,
			Options: []string{StatusDraft, StatusPublished},
		})
		collection.Workflow = &adminstudio.Workflow{
			StateField: StatusField,
			States:     []adminstudio.PublicationState{adminstudio.StateDraft, adminstudio.StatePublished},
			Transitions: map[adminstudio.PublicationState][]adminstudio.PublicationState{
				adminstudio.StateDraft:     {adminstudio.StatePublished},
				adminstudio.StatePublished: {adminstudio.StateDraft},
			},
		}
	}
	return collection
}

// Schemas and AdminCollections convert a set of definitions.
func Schemas(definitions []Definition) ([]collections.Schema, error) {
	result := make([]collections.Schema, 0, len(definitions))
	for _, definition := range definitions {
		schema, err := definition.Schema()
		if err != nil {
			return nil, err
		}
		result = append(result, schema)
	}
	return result, nil
}

// AdminCollections converts the definitions for the admin. With localize
// false (the site has one locale) localized fields are plain fields.
func AdminCollections(definitions []Definition, localize bool) []adminstudio.Collection {
	result := make([]adminstudio.Collection, 0, len(definitions))
	for _, definition := range definitions {
		collection := definition.AdminCollection()
		if !localize {
			collection.Localization = false
			for index := range collection.Fields {
				collection.Fields[index].Localized = false
			}
		}
		result = append(result, collection)
	}
	return result
}
