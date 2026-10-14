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
//	fields:
//	  - {id: question, label: Question, kind: text, required: true, list: true}
//	  - {id: answer, label: Answer, kind: richtext, required: true}
//	  - {id: category, label: Category, kind: select, options: [general, billing], list: true}
//	  - {id: owner, label: Owner, kind: relation, to: team}
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

// AdminBasePath is where the generic admin renders content types.
const AdminBasePath = "/admin/resources"

var ErrInvalidDefinition = errors.New("invalid content type definition")

// reserved field IDs are added by the storage layer to every record.
var reserved = map[string]bool{"id": true, "version": true, "created_at": true, "updated_at": true}

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// Definition is one content type as written in YAML.
type Definition struct {
	ID          string  `yaml:"id"`
	Label       string  `yaml:"label"`
	Description string  `yaml:"description"`
	Group       string  `yaml:"group"`
	Icon        string  `yaml:"icon"`
	TitleField  string  `yaml:"title_field"`
	Public      bool    `yaml:"public"`
	Fields      []Field `yaml:"fields"`
}

// Field kinds: text, textarea, richtext, number, boolean, date, datetime,
// select, media, relation (with to and optionally many).
type Field struct {
	ID       string   `yaml:"id"`
	Label    string   `yaml:"label"`
	Kind     string   `yaml:"kind"`
	Required bool     `yaml:"required"`
	Unique   bool     `yaml:"unique"`
	List     bool     `yaml:"list"`
	Options  []string `yaml:"options"`
	To       string   `yaml:"to"`
	Many     bool     `yaml:"many"`
	Min      *float64 `yaml:"min"`
	Max      *float64 `yaml:"max"`
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
		if field.Kind == "relation" && strings.TrimSpace(field.To) == "" {
			return fmt.Errorf("%w: relation field %q needs to:", ErrInvalidDefinition, field.ID)
		}
		if field.Kind != "relation" && (field.To != "" || field.Many) {
			return fmt.Errorf("%w: only relation fields take to/many (field %q)", ErrInvalidDefinition, field.ID)
		}
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
	schema := collections.Schema{ID: d.ID, Label: d.Label, Description: d.Description}
	for _, field := range d.Fields {
		kind, err := storageKind(field)
		if err != nil {
			return collections.Schema{}, err
		}
		schema.Fields = append(schema.Fields, collections.Field{
			ID: field.ID, Label: field.Label, Kind: kind, Required: field.Required, Unique: field.Unique,
			Min: field.Min, Max: field.Max, Options: append([]string(nil), field.Options...), Relation: field.To,
		})
	}
	return schema, nil
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

// AdminCollection returns the metadata the generic admin renders from.
func (d Definition) AdminCollection() adminstudio.Collection {
	collection := adminstudio.Collection{
		ID: d.ID, Label: d.Label, Description: d.Description, Group: d.Group, Icon: d.Icon,
		TitleField: d.TitleField, Versioned: true,
		Actions: []adminstudio.Action{{Label: "New " + d.Label, Href: AdminBasePath + "/" + d.ID + "/new"}},
	}
	if collection.Group == "" {
		collection.Group = "Content"
	}
	for _, field := range d.Fields {
		admin := adminstudio.Field{ID: field.ID, Label: field.Label, Required: field.Required, VisibleInList: field.List, Options: append([]string(nil), field.Options...)}
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

func AdminCollections(definitions []Definition) []adminstudio.Collection {
	result := make([]adminstudio.Collection, 0, len(definitions))
	for _, definition := range definitions {
		result = append(result, definition.AdminCollection())
	}
	return result
}
