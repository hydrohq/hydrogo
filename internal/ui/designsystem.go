package ui

import (
	"errors"
	"strings"
)

// DesignSystem is the small semantic bridge between HydroGo primitives and a
// visual provider. Providers map intent to classes; Core owns no provider CSS.
type DesignSystem interface {
	Name() string
	ButtonClass(variant string) string
	FieldClass() string
	TableClass() string
}

var ErrDesignSystemUnavailable = errors.New("design system unavailable")

type DesignSystemRegistry struct{ providers map[string]DesignSystem }

func NewDesignSystemRegistry(providers ...DesignSystem) (*DesignSystemRegistry, error) {
	r := &DesignSystemRegistry{providers: make(map[string]DesignSystem, len(providers))}
	for _, provider := range providers {
		if provider == nil || strings.TrimSpace(provider.Name()) == "" {
			return nil, ErrDesignSystemUnavailable
		}
		name := strings.ToLower(strings.TrimSpace(provider.Name()))
		if _, exists := r.providers[name]; exists {
			return nil, errors.New("duplicate design system: " + name)
		}
		r.providers[name] = provider
	}
	return r, nil
}

func (r *DesignSystemRegistry) Resolve(name string) (DesignSystem, error) {
	if r == nil {
		return nil, ErrDesignSystemUnavailable
	}
	provider, ok := r.providers[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, ErrDesignSystemUnavailable
	}
	return provider, nil
}

type SemanticDesignSystem struct{}

func (SemanticDesignSystem) Name() string { return "native" }
func (SemanticDesignSystem) ButtonClass(variant string) string {
	if strings.TrimSpace(variant) == "" {
		variant = "default"
	}
	return "button button-" + strings.TrimSpace(variant)
}
func (SemanticDesignSystem) FieldClass() string { return "form-field" }
func (SemanticDesignSystem) TableClass() string { return "ui-table" }

type ClassDesignSystem struct{ ID, Button, Field, Table string }

func (d ClassDesignSystem) Name() string { return d.ID }
func (d ClassDesignSystem) ButtonClass(variant string) string {
	if strings.TrimSpace(variant) == "" {
		variant = "default"
	}
	return strings.TrimSpace(d.Button) + " " + strings.TrimSpace(variant)
}
func (d ClassDesignSystem) FieldClass() string { return d.Field }
func (d ClassDesignSystem) TableClass() string { return d.Table }

func NewDefaultDesignSystemRegistry() (*DesignSystemRegistry, error) {
	return NewDesignSystemRegistry(SemanticDesignSystem{},
		ClassDesignSystem{ID: "shadcn", Button: "inline-flex items-center rounded-md", Field: "rounded-md border", Table: "w-full text-sm"},
		ClassDesignSystem{ID: "material", Button: "mdc-button", Field: "mdc-text-field", Table: "mdc-data-table"},
		ClassDesignSystem{ID: "carbon", Button: "cds--btn", Field: "cds--text-input", Table: "cds--data-table"})
}
