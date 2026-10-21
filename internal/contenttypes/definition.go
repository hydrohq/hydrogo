// Package contenttypes turns YAML content-type definitions into the two
// runtime shapes HydroGo needs: a collections.Schema for storage and
// validation, and an adminstudio.Collection for the generic admin screens.
//
// One file per type lives in the content-types directory:
//
//	id: faq
//	label: FAQ
//	title_field: question
//	public: true
//	drafts: true
//	revisions: true
//	fields:
//	  - {id: question, label: Question, kind: text, required: true, list: true, searchable: true}
//	  - {id: answer, label: Answer, kind: richtext, required: true, searchable: true}
//	  - {id: category, label: Category, kind: select, options: [general, billing], list: true, filterable: true}
//	  - {id: owner, label: Owner, kind: relation, to: team}
//
// Type keys: id, label, description, group, icon, title_field, fields, and:
//
//   - public: serve published records at /api/v1/content/{id}.
//   - drafts: give records a draft/published status (a "status" field the
//     admin changes with Publish and Move to draft). New records are drafts;
//     the public API never returns drafts.
//   - revisions: keep the last 50 saved versions of each record. The edit
//     page lists them; each can be compared with the current record and
//     restored (restoring saves a new version and keeps the current status).
//
// Field keys: id, label, kind, required, unique, options (select), to and
// many (relation), min and max (number), and:
//
//   - list: show the field as a column in the admin list. List columns are
//     sortable by clicking their heading.
//   - searchable: include the field in the admin list search and the public
//     API's ?search= (text, textarea, richtext, select).
//   - filterable: add a filter for the field to the admin list and allow
//     ?filter[field]= on the public API (text, textarea, select, number,
//     boolean, date, datetime and single relation fields). Beyond equality
//     the API accepts filter[field][ne|gt|gte|lt|lte|in|null]=value; gt, gte,
//     lt and lte need a number or date field, in takes a comma separated
//     list (at most 50 values) and null takes true or false.
//   - decimals: 2 fixes the decimal places of a number field (money). A
//     value with more places, or too large to be held exactly (more than 15
//     digits including the places), is rejected, never rounded.
//   - computed: "quantity * unit_price" calculates a number field on the
//     server on every save, with exact decimal arithmetic. It needs
//     decimals (the result is rounded half away from zero to that many
//     places), reads only number fields declared before it, and supports
//   - - * /, parentheses and abs, min, max and round(x, places). The
//     stored value is returned like any field, a client value is ignored,
//     and the field is read-only in the admin.
//   - sequence: {prefix: "INV-{yyyy}-", pad: 5}: number new records
//     without gaps ("INV-2026-00001", "INV-2026-00002", ...). The counter
//     restarts when the expanded prefix changes ({yyyy}, {yy} and {mm} are
//     replaced by the current UTC year and month). Text fields only; the
//     server sets the value on create, it never changes, and a value sent
//     by a client is ignored. The field is unique and read-only in the admin.
//   - localized: keep one value per site locale (text, textarea, richtext,
//     media; not together with unique, searchable or filterable). A type
//     with localized fields needs revisions: true. The record holds the
//     default locale's value; the admin edits the other locales through
//     its locale tabs, and the public API's ?locale= returns them, falling
//     back to the default locale per field. Localization is active only
//     when the site has i18n enabled with more than one locale; otherwise
//     the flag changes nothing.
//
// Rules and states (type level, next to fields):
//
//	rules:
//	  - assert: "quantity > 0"                 # every saved record must satisfy it
//	    message: "Quantity must be positive"
//	  - when: "status == 'posted'"             # ...only while the record is posted
//	    assert: "total > 0"
//	  - when: "status == 'posted'"             # a stored record matching it is locked:
//	    readonly: true                         # no edits, no delete
//	    message: "Posted invoices are locked"
//	states:
//	  field: status                            # a select field; read-only in the admin
//	  initial: draft                           # new records start here
//	  transitions:
//	    - {from: draft, to: approved, permission: invoice.approve}
//	    - {from: approved, to: posted, when: "total > 0", message: "Nothing to post"}
//	    - {from: posted, to: void}
//
// Conditions use the expression language of computed fields plus comparisons
// (== != < <= > >=), && || !, 'text', true, false and empty(field); they may
// only read fields of the type. A save can never change the state field: it
// moves along transitions with POST /api/v1/admin/content/{type}/{id}/state
// {"to": "approved", "version": n}, which needs the transition's permission
// (admins have every permission) and runs the asserts on the resulting
// record. Readonly rules do not block transitions, so a locked record can
// still be voided.
//
// The public API also accepts ?sort=field or ?sort=-field on any field and
// on created_at and updated_at.
package contenttypes

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/ardaozyildirim/hydrogo/internal/adminstudio"
	"github.com/ardaozyildirim/hydrogo/internal/collections"
)

var ErrInvalidDefinition = errors.New("invalid content type definition")

// reserved field IDs are added by the storage layer to every record.
// "locale" is added to public API responses.
var reserved = map[string]bool{"id": true, "version": true, "created_at": true, "updated_at": true, "locale": true}

var permissionName = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

// PermissionNames lists the custom permissions the definitions declare, for
// the permission catalog (type permissions and state-step permissions).
func PermissionNames(definitions []Definition) []string {
	seen := map[string]bool{}
	var names []string
	for _, d := range definitions {
		declared := []string{d.Permissions.Read, d.Permissions.Write}
		if d.Access != nil {
			declared = append(declared, d.Access.All)
		}
		for _, field := range d.Fields {
			declared = append(declared, field.Permission)
		}
		if d.States != nil {
			for _, step := range d.States.Transitions {
				declared = append(declared, step.Permission)
			}
		}
		for _, name := range declared {
			if name != "" && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// Definition is one content type as written in YAML.
type Definition struct {
	ID          string `json:"id" yaml:"id"`
	Label       string `json:"label" yaml:"label"`
	Description string `json:"description,omitempty" yaml:"description"`
	Group       string `json:"group,omitempty" yaml:"group"`
	Icon        string `json:"icon,omitempty" yaml:"icon"`
	TitleField  string `json:"title_field,omitempty" yaml:"title_field"`
	Public      bool   `json:"public" yaml:"public"`
	// Drafts adds a draft/published status to every record. New records
	// start as drafts, the admin publishes and unpublishes them, and the
	// public API only returns published records.
	Drafts bool `json:"drafts" yaml:"drafts"`
	// Revisions keeps the last collections.MaxRevisionsPerRecord saved
	// versions of each record; the admin can view and restore them.
	Revisions bool `json:"revisions" yaml:"revisions"`
	// Rules validate and lock records; States restricts how a select field
	// changes (see the package documentation).
	// Permissions names the permissions that guard this type in the admin API
	// instead of the generic content permissions.
	Access      *Access                   `json:"access,omitempty" yaml:"access"`
	Permissions Permissions               `json:"permissions,omitempty" yaml:"permissions"`
	Rules       []collections.Rule        `json:"rules,omitempty" yaml:"rules"`
	States      *collections.StateMachine `json:"states,omitempty" yaml:"states"`
	Fields      []Field                   `json:"fields" yaml:"fields"`
}

// Permissions guards one content type: Read for lists and reads, Write for
// creating, changing, deleting and state steps. An empty name keeps the
// generic admin.content.read / admin.content.write for that direction. Admins
// hold every permission; others get a named one from a per-user override.
type Permissions struct {
	Read  string `json:"read,omitempty" yaml:"read"`
	Write string `json:"write,omitempty" yaml:"write"`
}

// For returns the permission that guards reads or writes ("" = generic).
func (p Permissions) For(write bool) string {
	if write {
		return p.Write
	}
	return p.Read
}

// Field kinds: text, textarea, richtext, number, boolean, date, datetime,
// select, media, relation (with to and optionally many).
type Field struct {
	ID       string   `json:"id" yaml:"id"`
	Label    string   `json:"label" yaml:"label"`
	Kind     string   `json:"kind" yaml:"kind"`
	Required bool     `json:"required" yaml:"required"`
	Unique   bool     `json:"unique" yaml:"unique"`
	List     bool     `json:"list" yaml:"list"`
	Options  []string `json:"options,omitempty" yaml:"options"`
	To       string   `json:"to,omitempty" yaml:"to"`
	Many     bool     `json:"many,omitempty" yaml:"many"`
	// Permission hides the field from everyone without it: it is left out of
	// admin reads, ignored on writes (stored values stay), and cannot be
	// filtered, sorted, grouped or summed. Not allowed on required, unique,
	// searchable or localized fields.
	Permission string   `json:"permission,omitempty" yaml:"permission"`
	Min        *float64 `json:"min,omitempty" yaml:"min"`
	Max        *float64 `json:"max,omitempty" yaml:"max"`
	// Decimals fixes the decimal places of a number field (money: 2);
	// values with more places are rejected, never rounded.
	Decimals *int `json:"decimals,omitempty" yaml:"decimals"`
	// Computed calculates a number field on the server from other number
	// fields declared before it; see collections.Expr.
	Computed string `json:"computed,omitempty" yaml:"computed"`
	// Searchable fields (text, textarea, richtext, select) are matched by
	// the admin list search box and the public API's search parameter.
	Searchable bool `json:"searchable" yaml:"searchable"`
	// Filterable fields (see the package documentation) get a filter in
	// the admin list and accept filter[field]= on the public API.
	Filterable bool `json:"filterable" yaml:"filterable"`
	// Localized fields (text, textarea, richtext, media) have a value per
	// site locale; see the package documentation.
	Localized bool `json:"localized" yaml:"localized"`
	// Sequence numbers new records on the server (text fields only): see
	// collections.Sequence. The field is read-only in the admin.
	Sequence *collections.Sequence `json:"sequence,omitempty" yaml:"sequence"`
}

// LoadDir reads every *.yaml file in dir. A missing directory yields no
// definitions. Unknown YAML keys are rejected so typos fail loudly.
func LoadDir(dir string) ([]Definition, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	definitions := make([]Definition, 0, len(paths))
	seen := map[string]string{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var definition Definition
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		if err := decoder.Decode(&definition); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if err := definition.Validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if other, dup := seen[definition.ID]; dup {
			return nil, fmt.Errorf("%s: %w: id %q is already defined in %s", path, ErrInvalidDefinition, definition.ID, other)
		}
		seen[definition.ID] = path
		definitions = append(definitions, definition)
	}
	return definitions, nil
}

// Validate checks the definition and that both derived shapes are valid.
func (d Definition) Validate() error {
	if !identifier.MatchString(d.ID) {
		return fmt.Errorf("%w: id %q must be lowercase letters, digits and underscores", ErrInvalidDefinition, d.ID)
	}
	if strings.TrimSpace(d.Label) == "" {
		return fmt.Errorf("%w: label is required", ErrInvalidDefinition)
	}
	for _, field := range d.Fields {
		if reserved[field.ID] {
			return fmt.Errorf("%w: field id %q is reserved", ErrInvalidDefinition, field.ID)
		}
		if d.Drafts && field.ID == StatusField {
			return fmt.Errorf("%w: field id %q is reserved for drafts", ErrInvalidDefinition, field.ID)
		}
		if field.Kind == "relation" && strings.TrimSpace(field.To) == "" {
			return fmt.Errorf("%w: relation field %q needs to:", ErrInvalidDefinition, field.ID)
		}
		if field.Kind != "relation" && (field.To != "" || field.Many) {
			return fmt.Errorf("%w: only relation fields take to/many (field %q)", ErrInvalidDefinition, field.ID)
		}
		if field.Searchable && !oneOf(field.Kind, "text", "textarea", "richtext", "select") {
			return fmt.Errorf("%w: field %q of kind %s cannot be searchable", ErrInvalidDefinition, field.ID, field.Kind)
		}
		if field.Filterable && (!oneOf(field.Kind, "text", "textarea", "select", "number", "boolean", "date", "datetime", "relation") || field.Many) {
			return fmt.Errorf("%w: field %q of kind %s cannot be filterable", ErrInvalidDefinition, field.ID, field.Kind)
		}
		if field.Localized && !oneOf(field.Kind, "text", "textarea", "richtext", "media") {
			return fmt.Errorf("%w: field %q of kind %s cannot be localized", ErrInvalidDefinition, field.ID, field.Kind)
		}
		if field.Decimals != nil && (field.Kind != "number" || *field.Decimals < 0 || *field.Decimals > collections.MaxDecimals) {
			return fmt.Errorf("%w: decimals needs a number field and 0 to %d places (field %q)", ErrInvalidDefinition, collections.MaxDecimals, field.ID)
		}
		if field.Sequence != nil && (field.Kind != "text" || field.Required || field.Localized) {
			return fmt.Errorf("%w: sequence field %q must be a plain text field (not required or localized)", ErrInvalidDefinition, field.ID)
		}
		if field.Permission != "" {
			if !permissionName.MatchString(field.Permission) || strings.HasPrefix(field.Permission, "admin.") {
				return fmt.Errorf("%w: field %q permission %q must look like crm.margin.read", ErrInvalidDefinition, field.ID, field.Permission)
			}
			if d.Public {
				return fmt.Errorf("%w: a public type cannot have field permissions (field %q); the public API has no users", ErrInvalidDefinition, field.ID)
			}
			if field.Required || field.Unique || field.Searchable || field.Localized || field.Sequence != nil {
				return fmt.Errorf("%w: field %q has a permission, so it cannot be required, unique, searchable, localized or a sequence", ErrInvalidDefinition, field.ID)
			}
		}
		if field.Localized && (field.Unique || field.Searchable || field.Filterable) {
			return fmt.Errorf("%w: localized field %q cannot be unique, searchable or filterable", ErrInvalidDefinition, field.ID)
		}
	}
	for _, name := range []string{d.Permissions.Read, d.Permissions.Write} {
		if name != "" && (!permissionName.MatchString(name) || strings.HasPrefix(name, "admin.")) {
			return fmt.Errorf("%w: permission %q must look like crm.deals.read and not start with admin.", ErrInvalidDefinition, name)
		}
	}
	if d.Access != nil {
		owner, found := Field{}, false
		for _, field := range d.Fields {
			if field.ID == d.Access.Owner {
				owner, found = field, true
			}
		}
		if d.Public {
			return fmt.Errorf("%w: a type with access rules cannot be public", ErrInvalidDefinition)
		}
		if !found || owner.Kind != "text" || owner.Localized {
			return fmt.Errorf("%w: access.owner must name a plain text field", ErrInvalidDefinition)
		}
		if !permissionName.MatchString(d.Access.All) || strings.HasPrefix(d.Access.All, "admin.") {
			return fmt.Errorf("%w: access.all needs a permission such as deals.read_all", ErrInvalidDefinition)
		}
	}
	if d.Localized() && !d.Revisions {
		return fmt.Errorf("%w: a type with localized fields needs revisions: true", ErrInvalidDefinition)
	}
	schema, err := d.Schema()
	if err != nil {
		return err
	}
	if err := schema.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDefinition, err)
	}
	if _, err := adminstudio.NewRegistry([]adminstudio.Collection{d.AdminCollection()}); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDefinition, err)
	}
	return nil
}

// Schema returns the storage schema.
func (d Definition) Schema() (collections.Schema, error) {
	schema := collections.Schema{ID: d.ID, Label: d.Label, Description: d.Description, Revisions: d.Revisions, Rules: d.Rules, States: d.States}
	for _, field := range d.Fields {
		kind, err := storageKind(field)
		if err != nil {
			return collections.Schema{}, err
		}
		schema.Fields = append(schema.Fields, collections.Field{
			ID: field.ID, Label: field.Label, Kind: kind, Required: field.Required, Unique: field.Unique,
			Min: field.Min, Max: field.Max, Options: append([]string(nil), field.Options...), Relation: field.To,
			Localized: field.Localized, Indexed: field.Filterable, Sequence: field.Sequence, Decimals: field.Decimals, Computed: field.Computed,
		})
	}
	if d.Drafts {
		// Not required: records written before drafts were enabled have no
		// status and count as drafts (see recordStatus).
		schema.Fields = append(schema.Fields, collections.Field{ID: StatusField, Label: "Status", Kind: collections.Select, Options: []string{StatusDraft, StatusPublished}, Indexed: true})
	}
	return schema, nil
}

// Localized reports whether any field is localized.
func (d Definition) Localized() bool {
	for _, field := range d.Fields {
		if field.Localized {
			return true
		}
	}
	return false
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func storageKind(field Field) (collections.FieldKind, error) {
	switch field.Kind {
	case "text", "textarea":
		return collections.Text, nil
	case "richtext":
		return collections.RichText, nil
	case "number":
		return collections.Number, nil
	case "boolean":
		return collections.Boolean, nil
	case "date", "datetime":
		return collections.DateTime, nil
	case "select":
		return collections.Select, nil
	case "media":
		return collections.Media, nil
	case "relation":
		if field.Many {
			return collections.RelationMany, nil
		}
		return collections.Relation, nil
	default:
		return "", fmt.Errorf("%w: field %q has unknown kind %q", ErrInvalidDefinition, field.ID, field.Kind)
	}
}

// AdminCollection returns the metadata the admin renders its generic
// screens from.
func (d Definition) AdminCollection() adminstudio.Collection {
	collection := adminstudio.Collection{
		ID: d.ID, Label: d.Label, Description: d.Description, Group: d.Group, Icon: d.Icon,
		TitleField: d.TitleField, Versioned: true, Revisions: d.Revisions, Localization: d.Localized(),
		Source: adminstudio.SourceContentType, APIPath: AdminAPIPath + "/" + d.ID,
	}
	if collection.Group == "" {
		collection.Group = "Content"
	}
	if d.States != nil {
		machine := &adminstudio.StateMachine{Field: d.States.Field, Initial: d.States.Initial}
		for _, step := range d.States.Transitions {
			machine.Transitions = append(machine.Transitions, adminstudio.StateTransition{From: step.From, To: step.To, Permission: step.Permission})
		}
		collection.StateMachine = machine
	}
	for _, field := range d.Fields {
		admin := adminstudio.Field{
			ID: field.ID, Label: field.Label, Required: field.Required, VisibleInList: field.List,
			Searchable: field.Searchable, Filterable: field.Filterable, Options: append([]string(nil), field.Options...),
			Localized: field.Localized, ReadOnly: field.Sequence != nil || field.Computed != "" || (d.States != nil && d.States.Field == field.ID),
		}
		switch field.Kind {
		case "text":
			admin.Kind = adminstudio.FieldText
		case "textarea":
			admin.Kind, admin.Widget = adminstudio.FieldText, "textarea"
		case "richtext":
			admin.Kind = adminstudio.FieldRichText
		case "number":
			admin.Kind = adminstudio.FieldNumber
		case "boolean":
			admin.Kind = adminstudio.FieldBoolean
		case "date":
			admin.Kind = adminstudio.FieldDate
		case "datetime":
			admin.Kind, admin.Widget = adminstudio.FieldDate, "datetime-local"
		case "select":
			admin.Kind = adminstudio.FieldSelect
		case "media":
			admin.Kind = adminstudio.FieldMedia
		case "relation":
			admin.Kind = adminstudio.FieldRelation
			cardinality := "one"
			if field.Many {
				cardinality = "many"
			}
			admin.Relation = &adminstudio.RelationRef{Collection: field.To, Cardinality: cardinality}
		default:
			admin.Kind = adminstudio.FieldKind(field.Kind)
		}
		collection.Fields = append(collection.Fields, admin)
	}
	if d.Drafts {
		collection.Fields = append(collection.Fields, adminstudio.Field{
			ID: StatusField, Label: "Status", Kind: adminstudio.FieldSelect, VisibleInList: true, Filterable: true,
			Options: []string{StatusDraft, StatusPublished},
		})
		collection.Workflow = &adminstudio.Workflow{
			StateField: StatusField,
			States:     []adminstudio.PublicationState{adminstudio.StateDraft, adminstudio.StatePublished},
			Transitions: map[adminstudio.PublicationState][]adminstudio.PublicationState{
				adminstudio.StateDraft:     {adminstudio.StatePublished},
				adminstudio.StatePublished: {adminstudio.StateDraft},
			},
		}
	}
	return collection
}

// Schemas and AdminCollections convert a set of definitions.
func Schemas(definitions []Definition) ([]collections.Schema, error) {
	result := make([]collections.Schema, 0, len(definitions))
	for _, definition := range definitions {
		schema, err := definition.Schema()
		if err != nil {
			return nil, err
		}
		result = append(result, schema)
	}
	return result, nil
}

// AdminCollections converts the definitions for the admin. With localize
// false (the site has one locale) localized fields are plain fields.
func AdminCollections(definitions []Definition, localize bool) []adminstudio.Collection {
	result := make([]adminstudio.Collection, 0, len(definitions))
	for _, definition := range definitions {
		collection := definition.AdminCollection()
		if !localize {
			collection.Localization = false
			for index := range collection.Fields {
				collection.Fields[index].Localized = false
			}
		}
		result = append(result, collection)
	}
	return result
}
