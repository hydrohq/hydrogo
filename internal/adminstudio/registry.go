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
)

type Field struct {
	ID            string    `json:"id"`
	Label         string    `json:"label"`
	Kind          FieldKind `json:"kind"`
	Widget        string    `json:"widget,omitempty"`
	Required      bool      `json:"required"`
	ReadOnly      bool      `json:"read_only,omitempty"`
	VisibleInList bool      `json:"visible_in_list"`
	Options       []string  `json:"options,omitempty"`
}

type Action struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

// Collection is presentation metadata only. Mutations must still go through
// the module application service and its authorization policy.
type Collection struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Fields      []Field  `json:"fields"`
	Actions     []Action `json:"actions"`
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
	if strings.TrimSpace(collection.ID) == "" || strings.TrimSpace(collection.Label) == "" || len(collection.Fields) == 0 || len(collection.Actions) == 0 {
		return ErrInvalidCollection
	}
	seenFields := make(map[string]struct{}, len(collection.Fields))
	for _, field := range collection.Fields {
		if strings.TrimSpace(field.ID) == "" || strings.TrimSpace(field.Label) == "" || field.Kind == "" {
			return ErrInvalidCollection
		}
		if field.Widget != "" && field.Widget != "textarea" && field.Widget != "currency" {
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
	for _, action := range collection.Actions {
		if strings.TrimSpace(action.Label) == "" || !strings.HasPrefix(action.Href, "/") || strings.HasPrefix(action.Href, "//") {
			return ErrInvalidCollection
		}
	}
	return nil
}

func cloneCollection(collection Collection) Collection {
	clone := collection
	clone.Fields = make([]Field, len(collection.Fields))
	for i, field := range collection.Fields {
		clone.Fields[i] = field
		clone.Fields[i].Options = append([]string(nil), field.Options...)
	}
	clone.Actions = append([]Action(nil), collection.Actions...)
	return clone
}
