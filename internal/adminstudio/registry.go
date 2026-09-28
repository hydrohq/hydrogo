// Package adminstudio provides the metadata boundary used by generated admin
// surfaces. It describes a module without giving the UI direct database access.
package adminstudio

import (
	"errors"
	"sort"
	"strings"
)

var ErrInvalidCollection = errors.New("invalid admin collection")

type FieldKind string

const (
	FieldText     FieldKind = "text"
	FieldNumber   FieldKind = "number"
	FieldStatus   FieldKind = "status"
	FieldRelation FieldKind = "relation"
	FieldBoolean  FieldKind = "boolean"
	FieldDate     FieldKind = "date"
	FieldMedia    FieldKind = "media"
	FieldRichText FieldKind = "richtext"
	FieldSelect   FieldKind = "select"
	FieldArray    FieldKind = "array"
	FieldBlocks   FieldKind = "blocks"
)

// FieldRegistry is the provider-neutral field vocabulary used by generated
// admin metadata. A Client Edition can map these kinds to any widget system.
type FieldRegistry struct{ kinds map[FieldKind]struct{} }

func NewFieldRegistry() FieldRegistry {
	kinds := []FieldKind{FieldText, FieldNumber, FieldStatus, FieldRelation, FieldBoolean, FieldDate, FieldMedia, FieldRichText, FieldSelect, FieldArray, FieldBlocks}
	set := make(map[FieldKind]struct{}, len(kinds))
	for _, kind := range kinds {
		set[kind] = struct{}{}
	}
	return FieldRegistry{kinds: set}
}

func (r FieldRegistry) Supports(kind FieldKind) bool {
	_, ok := r.kinds[kind]
	return ok
}

type Field struct {
	ID            string    `json:"id"`
	Label         string    `json:"label"`
	Kind          FieldKind `json:"kind"`
	Widget        string    `json:"widget,omitempty"`
	Required      bool      `json:"required"`
	ReadOnly      bool      `json:"read_only,omitempty"`
	VisibleInList bool      `json:"visible_in_list"`
	Searchable    bool      `json:"searchable,omitempty"`
	Filterable    bool      `json:"filterable,omitempty"`
	Options       []string  `json:"options,omitempty"`
}

type Action struct {
	Label    string `json:"label"`
	Href     string `json:"href,omitempty"`
	CopyText string `json:"copy_text,omitempty"`
}

// Collection is presentation metadata only. Mutations must still go through
// the module application service and its authorization policy.
type Collection struct {
	ID          string   `json:"id"`
	Type        string   `json:"type,omitempty"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Fields      []Field  `json:"fields"`
	Actions     []Action `json:"actions"`
	BulkActions []Action `json:"bulk_actions,omitempty"`
	Kits        []Kit    `json:"kits,omitempty"`
}

type Registry struct {
	collections []Collection
}

func NewRegistry(collections []Collection) (Registry, error) {
	seen := make(map[string]struct{}, len(collections))
	copyCollections := append([]Collection(nil), collections...)
	for _, collection := range copyCollections {
		if err := validateCollection(collection); err != nil {
			return Registry{}, err
		}
		if _, exists := seen[collection.ID]; exists {
			return Registry{}, ErrInvalidCollection
		}
		seen[collection.ID] = struct{}{}
	}
	sort.Slice(copyCollections, func(i, j int) bool { return copyCollections[i].ID < copyCollections[j].ID })
	return Registry{collections: copyCollections}, nil
}

func (r Registry) Collections() []Collection {
	collections := make([]Collection, len(r.collections))
	for i, collection := range r.collections {
		collections[i] = cloneCollection(collection)
	}
	return collections
}

func (r Registry) Collection(id string) (Collection, bool) {
	for _, collection := range r.collections {
		if collection.ID == id {
			return cloneCollection(collection), true
		}
	}
	return Collection{}, false
}

func validateCollection(collection Collection) error {
	if strings.TrimSpace(collection.ID) == "" || (collection.Type != "" && collection.Type != "collection" && collection.Type != "global") || strings.TrimSpace(collection.Label) == "" || len(collection.Fields) == 0 || len(collection.Actions) == 0 {
		return ErrInvalidCollection
	}
	seenFields := make(map[string]struct{}, len(collection.Fields))
	fieldRegistry := NewFieldRegistry()
	for _, field := range collection.Fields {
		if strings.TrimSpace(field.ID) == "" || strings.TrimSpace(field.Label) == "" || !fieldRegistry.Supports(field.Kind) {
			return ErrInvalidCollection
		}
		if field.Widget != "" && !supportedWidget(field.Widget) {
			return ErrInvalidCollection
		}
		if _, exists := seenFields[field.ID]; exists {
			return ErrInvalidCollection
		}
		seenFields[field.ID] = struct{}{}
		seenOptions := make(map[string]struct{}, len(field.Options))
		for _, option := range field.Options {
			if strings.TrimSpace(option) == "" {
				return ErrInvalidCollection
			}
			if _, exists := seenOptions[option]; exists {
				return ErrInvalidCollection
			}
			seenOptions[option] = struct{}{}
		}
	}
	if err := ValidateKits(collection.Kits); err != nil {
		return ErrInvalidCollection
	}
	for _, action := range collection.Actions {
		if strings.TrimSpace(action.Label) == "" || !strings.HasPrefix(action.Href, "/") || strings.HasPrefix(action.Href, "//") {
			return ErrInvalidCollection
		}
	}
	for _, action := range collection.BulkActions {
		if strings.TrimSpace(action.Label) == "" || !strings.HasPrefix(action.Href, "/") || strings.HasPrefix(action.Href, "//") {
			return ErrInvalidCollection
		}
	}
	return nil
}

func supportedWidget(widget string) bool {
	switch widget {
	case "text", "textarea", "number", "boolean", "date", "datetime", "select", "media", "richtext", "block-editor", "relation", "array", "blocks", "currency", "checkbox", "hidden", "list":
		return true
	default:
		return false
	}
}

func cloneCollection(collection Collection) Collection {
	clone := collection
	clone.Fields = make([]Field, len(collection.Fields))
	for i, field := range collection.Fields {
		clone.Fields[i] = field
		clone.Fields[i].Options = append([]string(nil), field.Options...)
	}
	clone.Actions = append([]Action(nil), collection.Actions...)
	clone.BulkActions = append([]Action(nil), collection.BulkActions...)
	clone.Kits = append([]Kit(nil), collection.Kits...)
	return clone
}
