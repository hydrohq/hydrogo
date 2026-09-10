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
	ID            string
	Label         string
	Kind          FieldKind
	Required      bool
	VisibleInList bool
}

type Action struct {
	Label string
	Href  string
}

// Collection is presentation metadata only. Mutations must still go through
// the module application service and its authorization policy.
type Collection struct {
	ID          string
	Label       string
	Description string
	Fields      []Field
	Actions     []Action
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
	return append([]Collection(nil), r.collections...)
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
		if _, exists := seenFields[field.ID]; exists {
			return ErrInvalidCollection
		}
		seenFields[field.ID] = struct{}{}
	}
	for _, action := range collection.Actions {
		if strings.TrimSpace(action.Label) == "" || !strings.HasPrefix(action.Href, "/") || strings.HasPrefix(action.Href, "//") {
			return ErrInvalidCollection
		}
	}
	return nil
}
