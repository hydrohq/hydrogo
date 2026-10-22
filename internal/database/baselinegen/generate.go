package baselinegen

import (
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/database"
)

// GroupSpec is one baseline group: the migrations that need exactly Modules.
type GroupSpec struct {
	ID         string
	Modules    []string // sorted; empty for the core group
	Migrations []string // sorted migration IDs
}

// Generated is a group together with its rendered script.
type Generated struct {
	GroupSpec
	Script string
}

// ChainOnly hides the baseline of a bundle so the migration chain alone builds
// a database.
func ChainOnly(migrationFS fs.FS) fs.FS { return database.ChainOnly(migrationFS) }

// GroupID names the group for a module set: "core", the module, or "combo_a_b".
func GroupID(modules []string) string {
	switch len(modules) {
	case 0:
		return "core"
	case 1:
		return modules[0]
	default:
		return "combo_" + strings.Join(modules, "_")
	}
}

// ImpliedModules lists modules the chain needs without declaring it: forms
// extends the form_inbox table that the site module creates. The baseline makes
// the dependency explicit; the chain's own selection is unchanged.
var ImpliedModules = map[string][]string{"forms": {"site"}}

func withImplied(modules []string) []string {
	set := map[string]bool{}
	var visit func(string)
	visit = func(module string) {
		if set[module] {
			return
		}
		set[module] = true
		for _, implied := range ImpliedModules[module] {
			visit(implied)
		}
	}
	for _, module := range modules {
		visit(module)
	}
	out := make([]string, 0, len(set))
	for module := range set {
		out = append(out, module)
	}
	sort.Strings(out)
	return out
}

// Groups partitions the migrations by the modules they need. Core comes first,
// then smaller module sets, so a group's prerequisites precede it.
func Groups(migrationFS fs.FS) ([]GroupSpec, error) {
	needs, err := database.MigrationNeeds(migrationFS)
	if err != nil {
		return nil, err
	}
	byID := map[string]*GroupSpec{}
	for migration, modules := range needs {
		modules = withImplied(modules)
		id := GroupID(modules)
		group := byID[id]
		if group == nil {
			group = &GroupSpec{ID: id, Modules: modules}
			byID[id] = group
		}
		group.Migrations = append(group.Migrations, migration)
	}
	if byID["core"] == nil {
		return nil, fmt.Errorf("no migration is always enabled; a baseline needs a core group")
	}
	groups := make([]GroupSpec, 0, len(byID))
	for _, group := range byID {
		sort.Strings(group.Migrations)
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].Modules) != len(groups[j].Modules) {
			return len(groups[i].Modules) < len(groups[j].Modules)
		}
		return groups[i].ID < groups[j].ID
	})
	return groups, nil
}

func containsGroup(groups []GroupSpec, id string) bool {
	for _, group := range groups {
		if group.ID == id {
			return true
		}
	}
	return false
}

func isSubset(small, large []string) bool {
	if len(small) >= len(large) {
		return false
	}
	set := map[string]bool{}
	for _, module := range large {
		set[module] = true
	}
	for _, module := range small {
		if !set[module] {
			return false
		}
	}
	return true
}

// BuildChain applies the migration chain for a module set to an empty database
// from provider and snapshots the result.
func BuildChain(ctx context.Context, provider Provider, migrationFS fs.FS, modules []string) (*Snapshot, error) {
	db, cleanup, err := provider.New(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	enabled := map[string]bool{}
	for _, module := range modules {
		enabled[module] = true
	}
	if err := database.ApplyMigrationsForModules(ctx, db, provider.Dialect(), ChainOnly(migrationFS), enabled); err != nil {
		return nil, fmt.Errorf("apply the chain for %v: %w", modules, err)
	}
	return Take(ctx, db, provider.Dialect())
}

// Covered returns the groups a baseline replaces: all of them except chainOnly,
// whose migrations stay in the chain and run on top of the baseline.
func Covered(groups []GroupSpec, chainOnly []string) []GroupSpec {
	skip := map[string]bool{}
	for _, id := range chainOnly {
		skip[id] = true
	}
	var covered []GroupSpec
	for _, group := range groups {
		if !skip[group.ID] {
			covered = append(covered, group)
		}
	}
	return covered
}

// Generate renders one script per covered group for the provider's dialect.
// Each group's script is the difference between the chain built for its module
// set and the chains of the groups it builds on. Generate fails when a group
// would change something a prerequisite group owns, because a script cannot
// express that; such a group has to be named in chainOnly so leaving it to the
// chain is a decision, not an accident.
func Generate(ctx context.Context, provider Provider, migrationFS fs.FS, chainOnly []string) ([]Generated, error) {
	all, err := Groups(migrationFS)
	if err != nil {
		return nil, err
	}
	groups := Covered(all, chainOnly)
	for _, group := range groups {
		for _, other := range all {
			if isSubset(other.Modules, group.Modules) && !containsGroup(groups, other.ID) {
				return nil, fmt.Errorf("group %s builds on %s, which stays in the chain; it must stay there too", group.ID, other.ID)
			}
		}
	}
	snapshots := make(map[string]*Snapshot, len(groups))
	for _, group := range groups {
		snapshot, err := BuildChain(ctx, provider, migrationFS, group.Modules)
		if err != nil {
			return nil, fmt.Errorf("group %s: %w", group.ID, err)
		}
		snapshots[group.ID] = snapshot
	}
	var result []Generated
	for _, group := range groups {
		var below []*Snapshot
		for _, other := range groups {
			if isSubset(other.Modules, group.Modules) {
				below = append(below, snapshots[other.ID])
			}
		}
		script, err := render(provider.Dialect(), group, snapshots[group.ID], below)
		if err != nil {
			return nil, fmt.Errorf("group %s: %w", group.ID, err)
		}
		result = append(result, Generated{GroupSpec: group, Script: script})
	}
	return result, nil
}

func render(dialect database.Dialect, group GroupSpec, snapshot *Snapshot, below []*Snapshot) (string, error) {
	// What the groups below already provide.
	owned := map[string]Object{}
	ownedRows := map[string][]string{}
	for _, lower := range below {
		for _, object := range lower.Objects {
			if previous, ok := owned[object.Key()]; ok && canonical(previous) != canonical(object) {
				return "", fmt.Errorf("groups below disagree on %s", object.Key())
			}
			owned[object.Key()] = object
		}
		for table, rows := range lower.Rows {
			if previous, ok := ownedRows[table]; ok && strings.Join(previous, "\n") != strings.Join(rows, "\n") {
				return "", fmt.Errorf("groups below disagree on the rows of %s", table)
			}
			ownedRows[table] = rows
		}
	}
	var (
		drops, alters []string
		added         []Object
	)
	present := map[string]bool{}
	for _, object := range snapshot.Objects {
		present[object.Key()] = true
		previous, existed := owned[object.Key()]
		switch {
		case !existed:
			added = append(added, object)
		case canonical(previous) == canonical(object):
		case object.Kind == "table":
			statements, err := alterTable(dialect, previous, object)
			if err != nil {
				return "", err
			}
			alters = append(alters, statements...)
		case object.Kind == "function" || object.Kind == "rls":
			added = append(added, object) // CREATE OR REPLACE / ENABLE are repeatable
		default:
			statement, err := dropStatement(dialect, previous)
			if err != nil {
				return "", fmt.Errorf("changes %s that a group below owns: %w", object.Key(), err)
			}
			drops = append(drops, statement)
			added = append(added, object)
		}
	}
	var removed []Object
	for key, object := range owned {
		if !present[key] {
			removed = append(removed, object)
		}
	}
	sort.Slice(removed, func(i, j int) bool { return removed[i].Key() < removed[j].Key() })
	for _, object := range removed {
		statement, err := dropStatement(dialect, object)
		if err != nil {
			return "", err
		}
		drops = append(drops, statement)
	}

	addedTables := map[string]bool{}
	allTables := map[string]bool{}
	for _, object := range snapshot.Objects {
		if object.Kind == "table" {
			allTables[object.Name] = true
		}
	}
	for _, object := range added {
		if object.Kind == "table" {
			addedTables[object.Name] = true
		}
	}
	// Rows may only appear in tables the group creates.
	for table, rows := range snapshot.Rows {
		if addedTables[table] {
			continue
		}
		if strings.Join(rows, "\n") != strings.Join(ownedRows[table], "\n") {
			return "", fmt.Errorf("writes rows into %s that a group below owns", table)
		}
	}
	for table := range ownedRows {
		if !addedTables[table] && strings.Join(snapshot.Rows[table], "\n") != strings.Join(ownedRows[table], "\n") {
			return "", fmt.Errorf("removes rows from %s that a group below owns", table)
		}
	}
	// A table may only reference tables that exist in the group's own chain.
	for _, object := range added {
		for _, target := range referencedTables(object) {
			if !allTables[target] {
				return "", fmt.Errorf("%s references table %s, which no module it needs provides", object.Key(), target)
			}
		}
	}
	ordered, err := order(dialect, added)
	if err != nil {
		return "", err
	}
	var script strings.Builder
	fmt.Fprintf(&script, "-- Baseline %s (%s): the end state of %d migrations.\n-- Generated by tools/baselinegen; do not edit. See docs/decisions/0009.\n", group.ID, dialect, len(group.Migrations))
	for _, statement := range drops {
		script.WriteString("\n" + statement + "\n")
	}
	for _, statement := range alters {
		script.WriteString("\n" + statement + "\n")
	}
	for _, object := range ordered {
		script.WriteString("\n" + object.SQL + "\n")
	}
	var tables []string
	for table := range addedTables {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	for _, table := range tables {
		rows := snapshot.Rows[table]
		if len(rows) == 0 {
			continue
		}
		columns := make([]string, len(snapshot.Columns[table]))
		for index, column := range snapshot.Columns[table] {
			columns[index] = quoteIdent(dialect, column)
		}
		script.WriteString("\n")
		for _, tuple := range rows {
			fmt.Fprintf(&script, "INSERT INTO %s (%s) VALUES %s;\n", quoteIdent(dialect, table), strings.Join(columns, ", "), tuple)
		}
	}
	return script.String(), nil
}

var referenceTarget = regexp.MustCompile("(?i)REFERENCES\\s+[`\"\\[]?([A-Za-z0-9_]+)")

func referencedTables(object Object) []string {
	switch object.Kind {
	case "table", "foreign_key":
	default:
		return nil
	}
	var targets []string
	for _, match := range referenceTarget.FindAllStringSubmatch(object.SQL, -1) {
		targets = append(targets, match[1])
	}
	return targets
}

// kindRank orders PostgreSQL statements so every statement finds what it needs.
var kindRank = map[string]int{"table": 0, "constraint": 1, "index": 2, "foreign_key": 3, "function": 4, "trigger": 5, "rls": 6, "policy": 7, "view": 8}

func order(dialect database.Dialect, objects []Object) ([]Object, error) {
	ordered := append([]Object(nil), objects...)
	switch dialect {
	case database.Postgres:
		sort.SliceStable(ordered, func(i, j int) bool { return kindRank[ordered[i].Kind] < kindRank[ordered[j].Kind] })
		return ordered, nil
	case database.MySQL:
		return orderMySQL(ordered)
	default:
		return ordered, nil // SQLite: the catalog's own creation order
	}
}

// orderMySQL puts tables in foreign key order (MySQL checks the referenced table
// when a foreign key is created) and triggers last.
func orderMySQL(objects []Object) ([]Object, error) {
	var tables, rest []Object
	for _, object := range objects {
		if object.Kind == "table" {
			tables = append(tables, object)
		} else {
			rest = append(rest, object)
		}
	}
	byName := map[string]Object{}
	for _, table := range tables {
		byName[table.Name] = table
	}
	var out []Object
	state := map[string]int{} // 1 visiting, 2 done
	var visit func(name string) error
	visit = func(name string) error {
		switch state[name] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("foreign keys form a cycle through %s", name)
		}
		state[name] = 1
		for _, target := range referencedTables(byName[name]) {
			if target == name {
				continue
			}
			if _, ok := byName[target]; ok {
				if err := visit(target); err != nil {
					return err
				}
			}
		}
		state[name] = 2
		out = append(out, byName[name])
		return nil
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return append(out, rest...), nil
}
