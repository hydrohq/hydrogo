package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/ardaozyildirim/hydrogo/internal/collections"
	collectionstore "github.com/ardaozyildirim/hydrogo/internal/collections/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/generator"
	"github.com/ardaozyildirim/hydrogo/internal/pathguard"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

func collectionsCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || args[0] != "sync" {
		return errors.New("usage: hydro collections sync --confirm-admin-crud [--config config.yaml] [--generated generated] [--tenant ID]")
	}
	flags := flag.NewFlagSet("collections sync", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "config.yaml", "Edition configuration file")
	generatedRoot := flags.String("generated", "generated", "Generated Blueprint resource directory")
	tenantID := flags.String("tenant", "", "Tenant ID when syncing a tenant-scoped Edition")
	confirmAdminCRUD := flags.Bool("confirm-admin-crud", false, "Confirm these schemas will be editable by existing Admin Studio administrators")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if !*confirmAdminCRUD {
		return errors.New("sync requires --confirm-admin-crud after reviewing the generated Edition's authorization policy")
	}
	cfg, err := config.Load(*configPath, os.LookupEnv, config.Overrides{})
	if err != nil {
		return fmt.Errorf("load Edition configuration: %w", err)
	}
	if !enabledModuleSet(cfg.Modules)["collections"] {
		return errors.New("Blueprint sync requires the optional collections module to be enabled and migrated")
	}
	if cfg.Database.Dialect != database.SQLite && cfg.Database.Dialect != database.Postgres {
		return fmt.Errorf("Blueprint sync does not support database dialect %q", cfg.Database.Dialect)
	}
	ctx := context.Background()
	if cfg.Tenancy.Enabled {
		if strings.TrimSpace(*tenantID) == "" {
			return errors.New("--tenant is required when Edition tenancy is enabled")
		}
		if err := validateBlueprintSyncTenant(cfg.Tenancy.Hosts, *tenantID); err != nil {
			return err
		}
		scope, err := tenant.NewScope(*tenantID, *tenantID)
		if err != nil {
			return err
		}
		ctx = tenant.WithScope(ctx, scope)
	} else if strings.TrimSpace(*tenantID) != "" {
		return errors.New("--tenant can only be used when Edition tenancy is enabled")
	}

	schemas, err := loadBlueprintCollectionSchemas(*generatedRoot, cfg.Tenancy.Enabled)
	if err != nil {
		return err
	}
	db, err := database.Open(ctx, cfg.Database.Dialect, cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("open Edition database: %w", err)
	}
	defer db.Close()
	service := collections.Service{Repository: collectionstore.Store{
		DB: db, Dialect: cfg.Database.Dialect, RequireTenantScope: cfg.Tenancy.Enabled,
	}}
	created, unchanged, err := syncBlueprintCollectionSchemas(ctx, service, schemas)
	if err != nil {
		return fmt.Errorf("sync Blueprint collection schemas: %w (run hydro migrate first and review schema changes)", err)
	}
	_, err = fmt.Fprintf(stdout, "blueprint-collections-synced created=%d unchanged=%d admin=/admin/collections\n", created, unchanged)
	return err
}

func validateBlueprintSyncTenant(hosts map[string]string, tenantID string) error {
	for _, configuredID := range hosts {
		if configuredID == tenantID {
			return nil
		}
	}
	return errors.New("--tenant must match a tenant ID in config.yaml")
}

func loadBlueprintCollectionSchemas(root string, tenantEdition bool) ([]collections.Schema, error) {
	if err := pathguard.RejectSymlinkComponents(root); err != nil {
		return nil, fmt.Errorf("generated resource directory: %w", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("read generated resources: %w", err)
	}
	var schemas []collections.Schema
	seen := make(map[string]struct{})
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return nil, fmt.Errorf("inspect generated entry %q: %w", entry.Name(), err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("generated resource %q must not be a symlink", entry.Name())
		}
		if !info.IsDir() {
			continue
		}
		if err := pathguard.RejectSymlinkComponents(path); err != nil {
			return nil, fmt.Errorf("generated resource %q: %w", entry.Name(), err)
		}
		spec, err := generator.LoadSpec(filepath.Join(path, "spec.json"))
		if err != nil {
			return nil, fmt.Errorf("load Blueprint %q: %w", entry.Name(), err)
		}
		if spec.Admin.Hidden {
			continue
		}
		if spec.TenantScoped != tenantEdition {
			return nil, fmt.Errorf("Blueprint %q tenant scope must match the Edition tenancy configuration", spec.Name)
		}
		schema, err := blueprintCollectionSchema(spec)
		if err != nil {
			return nil, fmt.Errorf("Blueprint %q cannot use the dynamic collection runtime: %w", spec.Name, err)
		}
		if _, exists := seen[schema.ID]; exists {
			return nil, fmt.Errorf("duplicate Blueprint collection %q", schema.ID)
		}
		seen[schema.ID] = struct{}{}
		schemas = append(schemas, schema)
	}
	if len(schemas) == 0 {
		return nil, errors.New("no visible generated Blueprint resources found")
	}
	if len(schemas) > collections.MaxCollections {
		return nil, fmt.Errorf("Blueprint count exceeds dynamic collection limit (%d)", collections.MaxCollections)
	}
	sort.Slice(schemas, func(i, j int) bool { return schemas[i].ID < schemas[j].ID })
	return schemas, nil
}

func blueprintCollectionSchema(spec generator.Spec) (collections.Schema, error) {
	if spec.Type == "global" || spec.Auth || spec.Drafts || spec.Revisions || spec.Localization || spec.Workflow != nil {
		return collections.Schema{}, errors.New("global, protected, workflow, draft, revision, and localized resources require their generated Edition runtime")
	}
	if !hasGeneratedCRUDPermissions(spec.Name, spec.Permissions) {
		return collections.Schema{}, errors.New("the Blueprint must declare read, create, update, and delete permissions before Admin CRUD can be enabled")
	}
	label := spec.Admin.Label
	if strings.TrimSpace(label) == "" {
		label = humanizeBlueprintName(spec.Name)
	}
	schema := collections.Schema{ID: spec.Name, Label: label, Description: spec.Admin.Description}
	for _, source := range spec.Fields {
		if source.Admin.ReadOnly || source.Admin.Localized || source.Validation.Immutable || source.Validation.Default != nil || source.Validation.Regex != "" || source.Validation.Format != "" || source.Validation.MinLength != nil || source.Validation.MaxLength != nil {
			return collections.Schema{}, fmt.Errorf("field %q uses validation or presentation unsupported by the dynamic collection runtime", source.Name)
		}
		field := collections.Field{ID: source.Name, Label: source.Admin.Label, Required: !source.Nullable, Unique: source.Validation.Unique, Min: source.Validation.Min, Max: source.Validation.Max}
		if strings.TrimSpace(field.Label) == "" {
			field.Label = humanizeBlueprintName(source.Name)
		}
		typeName := strings.ToLower(strings.TrimSpace(source.Type))
		switch {
		case typeName == "relation" || typeName == "reference":
			if source.Relation == nil {
				return collections.Schema{}, fmt.Errorf("field %q has no relation target", source.Name)
			}
			if source.Relation.MaxItems != 0 {
				return collections.Schema{}, fmt.Errorf("field %q declares a relation item limit unsupported by the dynamic collection runtime", source.Name)
			}
			field.Relation = source.Relation.Collection
			if source.Relation.Cardinality == "many" {
				field.Kind = collections.RelationMany
			} else {
				field.Kind = collections.Relation
			}
		case len(source.Admin.Options) > 0 || typeName == "select" || typeName == "enum" || typeName == "status":
			field.Kind, field.Options = collections.Select, append([]string(nil), source.Admin.Options...)
		case typeName == "string" || typeName == "text" || typeName == "richtext" || typeName == "rich_text":
			if source.Admin.Widget == "richtext" || typeName == "richtext" || typeName == "rich_text" {
				field.Kind = collections.RichText
			} else {
				field.Kind = collections.Text
			}
		case typeName == "real" || typeName == "float" || typeName == "number":
			field.Kind = collections.Number
		case typeName == "bool" || typeName == "boolean":
			field.Kind = collections.Boolean
		case typeName == "datetime" || typeName == "timestamp":
			field.Kind = collections.DateTime
		case typeName == "media" || typeName == "upload" || typeName == "file":
			field.Kind = collections.Media
		default:
			return collections.Schema{}, fmt.Errorf("field %q type %q is unsupported", source.Name, source.Type)
		}
		schema.Fields = append(schema.Fields, field)
	}
	if err := schema.Validate(); err != nil {
		return collections.Schema{}, err
	}
	return schema, nil
}

func hasGeneratedCRUDPermissions(resourceID string, permissions []string) bool {
	read, write := false, false
	for _, permission := range permissions {
		switch permission {
		case resourceID + ".read":
			read = true
		case resourceID + ".write":
			write = true
		}
	}
	return resourceID != "" && read && write
}

type blueprintSchemaCatalog interface {
	ListSchemas(context.Context) ([]collections.Schema, error)
	CreateSchema(context.Context, collections.Schema) error
}

func syncBlueprintCollectionSchemas(ctx context.Context, catalog blueprintSchemaCatalog, desired []collections.Schema) (created, unchanged int, err error) {
	existing, err := catalog.ListSchemas(ctx)
	if err != nil {
		return 0, 0, err
	}
	byID := make(map[string]collections.Schema, len(existing))
	for _, schema := range existing {
		byID[schema.ID] = schema
	}
	var pending []collections.Schema
	for _, schema := range desired {
		if old, ok := byID[schema.ID]; ok {
			if !reflect.DeepEqual(old, schema) {
				return 0, 0, fmt.Errorf("Blueprint %q differs from its stored collection schema; review and update it in Admin Studio first", schema.ID)
			}
			unchanged++
			continue
		}
		pending = append(pending, schema)
	}
	if len(existing)+len(pending) > collections.MaxCollections {
		return 0, 0, fmt.Errorf("dynamic collection limit is %d", collections.MaxCollections)
	}
	ordered, err := orderBlueprintSchemas(pending, byID)
	if err != nil {
		return 0, 0, err
	}
	for _, schema := range ordered {
		if err := catalog.CreateSchema(ctx, schema); err != nil {
			return created, unchanged, fmt.Errorf("create schema %q: %w", schema.ID, err)
		}
		created++
	}
	return created, unchanged, nil
}

func orderBlueprintSchemas(pending []collections.Schema, existing map[string]collections.Schema) ([]collections.Schema, error) {
	byID := make(map[string]collections.Schema, len(pending))
	for _, schema := range pending {
		byID[schema.ID] = schema
	}
	state := make(map[string]uint8, len(pending))
	ordered := make([]collections.Schema, 0, len(pending))
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case 1:
			return fmt.Errorf("Blueprint relation cycle includes %q", id)
		case 2:
			return nil
		}
		schema, ok := byID[id]
		if !ok {
			if _, found := existing[id]; !found {
				return fmt.Errorf("Blueprint relation target %q is not synced", id)
			}
			return nil
		}
		state[id] = 1
		for _, field := range schema.Fields {
			if field.Kind == collections.Relation || field.Kind == collections.RelationMany {
				if field.Relation == id {
					continue
				}
				if err := visit(field.Relation); err != nil {
					return err
				}
			}
		}
		state[id] = 2
		ordered = append(ordered, schema)
		return nil
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

func humanizeBlueprintName(value string) string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "_", " "), "-", " ")
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
