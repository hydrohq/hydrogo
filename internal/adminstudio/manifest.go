package adminstudio

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

var ErrInvalidManifest = errors.New("invalid generated admin manifest")

// ParseManifest accepts only the current, bounded Admin metadata contract.
func ParseManifest(data []byte) (Registry, error) {
	if len(data) == 0 || len(data) > 1<<20 {
		return Registry{}, ErrInvalidManifest
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil || manifest.Version != "v1" || len(manifest.Resources) == 0 {
		return Registry{}, ErrInvalidManifest
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return Registry{}, ErrInvalidManifest
	}
	registry, err := NewRegistry(manifest.Resources)
	if err != nil {
		return Registry{}, ErrInvalidManifest
	}
	return registry, nil
}
