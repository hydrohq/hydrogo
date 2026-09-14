package ui

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// FrontendManifest is the stable, versioned contract between generated HTML,
// browser behavior, and optional external frontend clients.
type FrontendManifest struct {
	Version   int                `json:"version"`
	Routes    []FrontendRoute    `json:"routes"`
	Selectors []FrontendSelector `json:"selectors"`
}

type FrontendRoute struct {
	Name   string `json:"name"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Access string `json:"access"`
}

type FrontendSelector struct {
	Name     string `json:"name"`
	Selector string `json:"selector"`
	Purpose  string `json:"purpose"`
}

var ErrInvalidFrontendManifest = errors.New("invalid frontend manifest")

// Validate rejects ambiguous or unversioned contracts before they are
// published to a generated project or consumed by a browser client.
func (m FrontendManifest) Validate() error {
	if m.Version < 1 {
		return fmt.Errorf("%w: version must be positive", ErrInvalidFrontendManifest)
	}
	routeNames := map[string]struct{}{}
	for _, route := range m.Routes {
		if route.Name == "" || route.Path == "" || !strings.HasPrefix(route.Path, "/") || route.Method == "" {
			return fmt.Errorf("%w: route name, method, and absolute path are required", ErrInvalidFrontendManifest)
		}
		if route.Method != http.MethodGet && route.Method != http.MethodPost && route.Method != http.MethodPut && route.Method != http.MethodPatch && route.Method != http.MethodDelete {
			return fmt.Errorf("%w: unsupported method %q", ErrInvalidFrontendManifest, route.Method)
		}
		if route.Access != "public" && route.Access != "protected" {
			return fmt.Errorf("%w: unsupported route access %q", ErrInvalidFrontendManifest, route.Access)
		}
		if _, exists := routeNames[route.Name]; exists {
			return fmt.Errorf("%w: duplicate route %q", ErrInvalidFrontendManifest, route.Name)
		}
		routeNames[route.Name] = struct{}{}
	}
	selectorNames := map[string]struct{}{}
	for _, selector := range m.Selectors {
		if selector.Name == "" || selector.Selector == "" || selector.Purpose == "" {
			return fmt.Errorf("%w: selector name, selector, and purpose are required", ErrInvalidFrontendManifest)
		}
		if _, exists := selectorNames[selector.Name]; exists {
			return fmt.Errorf("%w: duplicate selector %q", ErrInvalidFrontendManifest, selector.Name)
		}
		selectorNames[selector.Name] = struct{}{}
	}
	return nil
}
