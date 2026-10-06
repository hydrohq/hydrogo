// Package adminstudio is the versioned public composition contract for
// Blueprint-generated Admin surfaces in Client Editions.
//
// This package deliberately exposes the runtime's typed metadata and handler
// options without granting authorization, tenant scope, migration, or route
// activation policy. Editions must supply those boundaries explicitly.
package adminstudio

import (
	"errors"
	"net/http"
	"strings"

	internal "github.com/ardaozyildirim/hydrogo/internal/adminstudio"
)

type (
	Action                = internal.Action
	Collection            = internal.Collection
	DraftDocument         = internal.DraftDocument
	Field                 = internal.Field
	FieldKind             = internal.FieldKind
	GeneratedAdminHandler = internal.GeneratedAdminHandler
	GeneratedAdminOptions = internal.GeneratedAdminOptions
	LocaleOptions         = internal.LocaleOptions
	PublicationState      = internal.PublicationState
	Registry              = internal.Registry
	RelationRef           = internal.RelationRef
	RevisionSnapshot      = internal.RevisionSnapshot
	RevisionSummary       = internal.RevisionSummary
	Workflow              = internal.Workflow
)

const (
	FieldText      = internal.FieldText
	FieldRichText  = internal.FieldRichText
	FieldSelect    = internal.FieldSelect
	FieldStatus    = internal.FieldStatus
	FieldRelation  = internal.FieldRelation
	FieldMedia     = internal.FieldMedia
	StateDraft     = internal.StateDraft
	StateScheduled = internal.StateScheduled
	StatePublished = internal.StatePublished
	StateRetracted = internal.StateRetracted
)

var (
	NewRegistry   = internal.NewRegistry
	ParseManifest = internal.ParseManifest
)

// RegisterGenerated parses the generated Edition manifest and registers only
// the explicitly selected resources using caller-supplied security, API,
// CSRF, and page-shell boundaries. Calling this function is the activation
// step; an empty or invalid resource selection fails closed.
func RegisterGenerated(mux *http.ServeMux, manifest []byte, activeResources []string, options GeneratedAdminOptions) error {
	registry, err := ParseManifest(manifest)
	if err != nil {
		return err
	}
	collections := registry.Collections()
	if len(activeResources) == 0 || len(activeResources) > len(collections) {
		return errors.New("generated Admin activation requires a valid resource selection")
	}
	active := make(map[string]bool, len(activeResources))
	for _, id := range activeResources {
		id = strings.TrimSpace(id)
		if id == "" || active[id] {
			return errors.New("generated Admin activation requires a valid resource selection")
		}
		active[id] = true
	}
	filtered := internal.FilterActiveCollections(collections, active)
	if len(filtered) != len(activeResources) {
		return errors.New("generated Admin activation references an unknown resource")
	}
	registry, err = internal.NewRegistry(filtered)
	if err != nil {
		return err
	}
	return (GeneratedAdminHandler{Registry: registry, Options: options}).Register(mux)
}
