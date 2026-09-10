package generator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SpecDocument is the versioned, file-backed form of a module specification.
// New schema versions must be introduced deliberately instead of being inferred
// from fields that happen to be present in a source file.
type SpecDocument struct {
	Version    string `json:"version" yaml:"version"`
	Collection Spec   `json:"collection" yaml:"collection"`
}

// LoadSpec reads a JSON or YAML module spec. It rejects symbolic links and
// unknown YAML fields so an automation run cannot silently compile a different
// contract than the one reviewers approved.
func LoadSpec(path string) (Spec, error) {
	if strings.TrimSpace(path) == "" {
		return Spec{}, ErrInvalidSpec
	}
	info, err := os.Lstat(path)
	if err != nil {
		return Spec{}, fmt.Errorf("read module spec: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Spec{}, fmt.Errorf("module spec must be a real regular file: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Spec{}, fmt.Errorf("read module spec: %w", err)
	}
	var document SpecDocument
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&document)
	case ".yaml", ".yml":
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		err = decoder.Decode(&document)
	default:
		return Spec{}, errors.New("module spec must use .json, .yaml, or .yml")
	}
	if err != nil {
		return Spec{}, fmt.Errorf("decode module spec: %w", err)
	}
	if document.Version != "v0" {
		return Spec{}, fmt.Errorf("unsupported module spec version %q", document.Version)
	}
	if err := Validate(document.Collection); err != nil {
		return Spec{}, err
	}
	return document.Collection, nil
}
