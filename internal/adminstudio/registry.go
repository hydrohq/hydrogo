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
	ID            string       `json:"id"`
	Label         string       `json:"label"`
	Kind          FieldKind    `json:"kind"`
	Widget        string       `json:"widget,omitempty"`
	Required      bool         `json:"required"`
	ReadOnly      bool         `json:"read_only,omitempty"`
	VisibleInList bool         `json:"visible_in_list"`
	Searchable    bool         `json:"searchable,omitempty"`
	Filterable    bool         `json:"filterable,omitempty"`
	Options       []string     `json:"options,omitempty"`
	Relation      *RelationRef `json:"relation,omitempty"`
}

type RelationRef struct {
	Collection  string `json:"collection"`
	Cardinality string `json:"cardinality"`
	MaxItems    int    `json:"max_items,omitempty"`
}

type Action struct {
	Label    string `json:"label"`
	Href     string `json:"href,omitempty"`
	CopyText string `json:"copy_text,omitempty"`
}

// Collection is presentation metadata only. Mutations must still go through
// the module application service and its authorization policy.
type Collection struct {
	ID          string    `json:"id"`
	Type        string    `json:"type,omitempty"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
	Group       string    `json:"group,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	Hidden      bool      `json:"hidden,omitempty"`
	TitleField  string    `json:"title_field,omitempty"`
	Workflow    *Workflow `json:"workflow,omitempty"`
	Fields      []Field   `json:"fields"`
	Actions     []Action  `json:"actions"`
	BulkActions []Action  `json:"bulk_actions,omitempty"`
	Kits        []Kit     `json:"kits,omitempty"`
}

type Registry struct {
	collections []Collection
}

// Manifest is the build-time registry consumed by Edition Admin composition.
type Manifest struct {
	Version   string       `json:"version"`
	Resources []Collection `json:"resources"`
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
	titleFieldFound := collection.TitleField == ""
	for _, field := range collection.Fields {
		if strings.TrimSpace(field.ID) == "" || strings.TrimSpace(field.Label) == "" || !fieldRegistry.Supports(field.Kind) {
			return ErrInvalidCollection
		}
		if field.Widget != "" && !supportedWidget(field.Widget) {
			return ErrInvalidCollection
		}
		if field.Kind == FieldRelation && field.Relation != nil && (field.Relation.Collection == "" || (field.Relation.Cardinality != "one" && field.Relation.Cardinality != "many") || (field.Relation.Cardinality == "one" && field.Relation.MaxItems != 0) || (field.Relation.Cardinality == "many" && (field.Relation.MaxItems < 0 || field.Relation.MaxItems > 100))) {
			return ErrInvalidCollection
		}
		if _, exists := seenFields[field.ID]; exists {
			return ErrInvalidCollection
		}
		seenFields[field.ID] = struct{}{}
		if field.ID == collection.TitleField {
			titleFieldFound = true
		}
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
	if !titleFieldFound {
		return ErrInvalidCollection
	}
	if collection.Workflow != nil {
		workflow := collection.Workflow
		if workflow.StateField == "" || workflow.Validate() != nil {
			return ErrInvalidCollection
		}
		stateField, found := func() (Field, bool) {
			for _, field := range collection.Fields {
				if field.ID == workflow.StateField {
					return field, true
				}
			}
			return Field{}, false
		}()
		if !found || (stateField.Kind != FieldStatus && stateField.Kind != FieldSelect && stateField.Widget != "select") {
			return ErrInvalidCollection
		}
		if len(stateField.Options) > 0 {
			allowed := make(map[string]struct{}, len(stateField.Options))
			for _, option := range stateField.Options {
				allowed[option] = struct{}{}
			}
			for _, state := range workflow.States {
				if _, ok := allowed[string(state)]; !ok {
					return ErrInvalidCollection
				}
			}
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
	case "text", "textarea", "number", "boolean", "date", "datetime", "datetime-local", "select", "media", "richtext", "block-editor", "relation", "array", "blocks", "currency", "checkbox", "hidden", "list":
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
		if field.Relation != nil {
			relation := *field.Relation
			clone.Fields[i].Relation = &relation
		}
	}
	clone.Actions = append([]Action(nil), collection.Actions...)
	clone.BulkActions = append([]Action(nil), collection.BulkActions...)
	clone.Kits = append([]Kit(nil), collection.Kits...)
	if collection.Workflow != nil {
		workflow := *collection.Workflow
		workflow.States = append([]PublicationState(nil), collection.Workflow.States...)
		workflow.Transitions = make(map[PublicationState][]PublicationState, len(collection.Workflow.Transitions))
		for from, to := range collection.Workflow.Transitions {
			workflow.Transitions[from] = append([]PublicationState(nil), to...)
		}
		clone.Workflow = &workflow
	}
	return clone
}
