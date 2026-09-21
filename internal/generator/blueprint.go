package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Blueprint is the ergonomic authoring format for a generated collection.
// It is deliberately translated into the reviewed Spec contract before any
// artifact is written.
type Blueprint struct {
	Name   string           `json:"name" yaml:"name"`
	Table  string           `json:"table,omitempty" yaml:"table,omitempty"`
	Fields []BlueprintField `json:"fields" yaml:"fields"`
	Admin  Admin            `json:"admin,omitempty" yaml:"admin,omitempty"`
	Kits   []string         `json:"kits,omitempty" yaml:"kits,omitempty"`
}

type BlueprintField struct {
	Name     string   `json:"name" yaml:"name"`
	Type     string   `json:"type" yaml:"type"`
	Required bool     `json:"required" yaml:"required"`
	Values   []string `json:"values,omitempty" yaml:"values,omitempty"`
}

func LoadBlueprint(path string) (Spec, error) {
	if strings.TrimSpace(path) == "" {
		return Spec{}, ErrInvalidSpec
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Spec{}, fmt.Errorf("blueprint must be a regular file: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Spec{}, fmt.Errorf("read blueprint: %w", err)
	}
	var blueprint Blueprint
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&blueprint)
	case ".yaml", ".yml":
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		err = decoder.Decode(&blueprint)
	default:
		return Spec{}, fmt.Errorf("blueprint must use .yaml, .yml, or .json")
	}
	if err != nil {
		return Spec{}, fmt.Errorf("decode blueprint: %w", err)
	}
	name := strings.ToLower(strings.TrimSpace(blueprint.Name))
	table := strings.ToLower(strings.TrimSpace(blueprint.Table))
	if table == "" {
		table = name + "s"
	}
	fields := make([]Field, 0, len(blueprint.Fields))
	for _, field := range blueprint.Fields {
		typ := strings.ToLower(strings.TrimSpace(field.Type))
		admin := FieldAdmin{}
		if len(field.Values) > 0 {
			typ = "string"
			admin.Widget = "select"
		}
		fields = append(fields, Field{Name: strings.ToLower(strings.TrimSpace(field.Name)), Type: typ, Nullable: !field.Required, Admin: admin})
	}
	permissions := []string{name + ".read", name + ".write"}
	spec := Spec{Name: name, Table: table, Fields: fields, Permissions: permissions, Dialects: []string{"sqlite", "postgres"}, Admin: blueprint.Admin, Kits: blueprint.Kits}
	if err := Validate(spec); err != nil {
		return Spec{}, err
	}
	return spec, nil
}
