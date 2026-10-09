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

	"github.com/a-h/templ"
	internal "github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	internalAuth "github.com/ardaozyildirim/hydrogo/internal/auth"
	internalUI "github.com/ardaozyildirim/hydrogo/internal/ui"
)

type (
	PageRole              string
	PageIdentityResolver  func(*http.Request) PageIdentity
	Action                = internal.Action
	BlockEditorConfig     = internal.BlockEditorConfig
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
	ResourceAccess        = internal.ResourceAccess
	Workflow              = internal.Workflow
)

// PageIdentity is presentation data for the shared Admin Studio shell. It is
// not an authorization decision; GeneratedAdminOptions.Authorize remains the
// route boundary, CanReadResource filters readable resources, and
// GeneratedAdminOptions.ResourceAccess controls basic CRUD affordances. Admin,
// editor, and viewer are the supported presentation roles.
type PageIdentity struct {
	UserID string
	Email  string
	Role   PageRole
}

const (
	PageRoleAdmin  PageRole = "admin"
	PageRoleEditor PageRole = "editor"
	PageRoleViewer PageRole = "viewer"
	FieldText               = internal.FieldText
	FieldRichText           = internal.FieldRichText
	FieldSelect             = internal.FieldSelect
	FieldStatus             = internal.FieldStatus
	FieldRelation           = internal.FieldRelation
	FieldMedia              = internal.FieldMedia
	FieldArray              = internal.FieldArray
	FieldBlocks             = internal.FieldBlocks
	StateDraft              = internal.StateDraft
	StateScheduled          = internal.StateScheduled
	StatePublished          = internal.StatePublished
	StateRetracted          = internal.StateRetracted
)

// StudioPageRenderer adapts an Edition's authenticated identity to the shared
// HydroGo Admin Studio shell, so independently built Editions can reuse the
// same layout without importing Core's internal UI package. The presentation
// role never grants access; Edition authorization and resource policies do.
func StudioPageRenderer(resolve PageIdentityResolver) func(http.ResponseWriter, *http.Request, string, templ.Component) error {
	return func(w http.ResponseWriter, r *http.Request, title string, body templ.Component) error {
		if resolve == nil || r == nil || body == nil {
			return errors.New("Admin Studio page rendering requires request, identity and content")
		}
		identity := resolve(r)
		role := internalAuth.Role(identity.Role)
		if strings.TrimSpace(identity.UserID) == "" || (role != internalAuth.RoleAdmin && role != internalAuth.RoleEditor && role != internalAuth.RoleViewer) {
			return errors.New("Admin Studio page identity is invalid")
		}
		pageRequest := r.WithContext(internalAuth.WithUser(r.Context(), &internalAuth.User{
			ID: identity.UserID, Email: identity.Email, Role: role, Active: true,
		}))
		if err := internalUI.AdminStartForRequest(w, pageRequest, title, "", "generated"); err != nil {
			return err
		}
		if err := body.Render(pageRequest.Context(), w); err != nil {
			return err
		}
		return internalUI.AdminEnd(w)
	}
}

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
