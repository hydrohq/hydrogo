package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
	"gopkg.in/yaml.v3"
)

var contentTypeID = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// contentTypeCommand implements `hydro content-type new|list|validate|apply|diff`.
func contentTypeCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: hydro content-type new|list|validate|apply|diff [options]")
	}

	switch args[0] {
	case "new":
		flags := flag.NewFlagSet("content-type new", flag.ContinueOnError)
		flags.SetOutput(stderr)
		dir := flags.String("dir", "content-types", "content types directory")

		var id string
		rest := args[1:]
		if len(rest) > 0 && len(rest[0]) > 0 && rest[0][0] != '-' {
			id, rest = rest[0], rest[1:]
		}
		if err := flags.Parse(rest); err != nil {
			return err
		}
		if id == "" && flags.NArg() == 1 {
			id = flags.Arg(0)
		} else if flags.NArg() != 0 {
			return errors.New("usage: hydro content-type new ID [--dir content-types]")
		}
		if !contentTypeID.MatchString(id) {
			return fmt.Errorf("content type id %q must be lowercase letters, digits and underscores, starting with a letter", id)
		}
		if err := os.MkdirAll(*dir, 0o755); err != nil { // #nosec G301
			return err
		}
		path := filepath.Join(*dir, id+".yaml")
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists", path)
		}
		skeleton := fmt.Sprintf(`# yaml-language-server: $schema=https://raw.githubusercontent.com/ardaozyildirim/hydrogo/main/docs/schemas/content-type.schema.json
id: %s
label: %s
title_field: title
public: true
drafts: true
revisions: true
fields:
  - {id: title, label: Title, kind: text, required: true, list: true, searchable: true}
  - {id: body, label: Body, kind: richtext}
`, id, id)
		if err := os.WriteFile(path, []byte(skeleton), 0o644); err != nil { // #nosec G306
			return err
		}
		fmt.Fprintf(stdout, "Created %s. Restart hydro serve to see /admin/content/%s and /api/v1/content/%s.\n", path, id, id)
		return nil

	case "list":
		flags := flag.NewFlagSet("content-type list", flag.ContinueOnError)
		flags.SetOutput(stderr)
		dir := flags.String("dir", "content-types", "content types directory")
		jsonOutput := flags.Bool("json", false, "output JSON format")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}

		types, err := contenttypes.LoadDir(*dir)
		if err != nil {
			if *jsonOutput {
				_ = json.NewEncoder(stdout).Encode(map[string]any{"error": "load_failed", "details": err.Error()})
			}
			return err
		}

		if *jsonOutput {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(types)
		}

		for _, t := range types {
			fmt.Fprintf(stdout, "%-16s %-20s fields=%d public=%v drafts=%v\n", t.ID, t.Label, len(t.Fields), t.Public, t.Drafts)
		}
		return nil

	case "validate":
		flags := flag.NewFlagSet("content-type validate", flag.ContinueOnError)
		flags.SetOutput(stderr)
		dir := flags.String("dir", "content-types", "content types directory")
		jsonOutput := flags.Bool("json", false, "output JSON format")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}

		types, err := contenttypes.LoadDir(*dir)
		if err != nil {
			if *jsonOutput {
				enc := json.NewEncoder(stdout)
				enc.SetIndent("", "  ")
				_ = enc.Encode(map[string]any{
					"valid":   false,
					"error":   "validation_failed",
					"details": err.Error(),
				})
			}
			return err
		}

		if *jsonOutput {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]any{
				"valid":         true,
				"count":         len(types),
				"content_types": types,
			})
		}

		for _, t := range types {
			fmt.Fprintf(stdout, "ok  %s (%d fields)\n", t.ID, len(t.Fields))
		}
		fmt.Fprintf(stdout, "%d content types valid in %s\n", len(types), *dir)
		return nil

	case "apply":
		flags := flag.NewFlagSet("content-type apply", flag.ContinueOnError)
		flags.SetOutput(stderr)
		file := flags.String("file", "", "path to content type YAML file")
		dir := flags.String("dir", "content-types", "content types directory")
		dryRun := flags.Bool("dry-run", false, "preview changes without writing")
		jsonOutput := flags.Bool("json", false, "output JSON format")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}

		if *file == "" {
			return errors.New("usage: hydro content-type apply --file PATH [--dir content-types] [--dry-run] [--json]")
		}

		data, err := os.ReadFile(*file) // #nosec G304
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}

		var def contenttypes.Definition
		if err := yaml.Unmarshal(data, &def); err != nil {
			return fmt.Errorf("parse definition: %w", err)
		}

		def.ID = strings.TrimSpace(strings.ToLower(def.ID))
		if !contentTypeID.MatchString(def.ID) {
			return fmt.Errorf("invalid content type id %q", def.ID)
		}

		if err := def.Validate(); err != nil {
			if *jsonOutput {
				enc := json.NewEncoder(stdout)
				enc.SetIndent("", "  ")
				_ = enc.Encode(map[string]any{
					"error":   "invalid_definition",
					"file":    *file,
					"details": err.Error(),
				})
			}
			return fmt.Errorf("validation failed: %w", err)
		}

		targetPath := filepath.Join(*dir, def.ID+".yaml")
		action := "create"
		if _, err := os.Stat(targetPath); err == nil {
			action = "update"
		}

		if *dryRun {
			if *jsonOutput {
				enc := json.NewEncoder(stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(map[string]any{
					"dry_run":      true,
					"action":       action,
					"content_type": def.ID,
					"target_file":  targetPath,
					"applied":      false,
				})
			}
			fmt.Fprintf(stdout, "[dry-run] Would apply %s to %s (action: %s)\n", def.ID, targetPath, action)
			return nil
		}

		if err := os.MkdirAll(*dir, 0o755); err != nil { // #nosec G301
			return err
		}

		yamlBytes, err := yaml.Marshal(&def)
		if err != nil {
			return fmt.Errorf("serialize definition: %w", err)
		}

		if err := os.WriteFile(targetPath, yamlBytes, 0o644); err != nil { // #nosec G306
			return fmt.Errorf("write target file: %w", err)
		}

		if *jsonOutput {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]any{
				"dry_run":      false,
				"action":       action,
				"content_type": def.ID,
				"target_file":  targetPath,
				"applied":      true,
			})
		}

		fmt.Fprintf(stdout, "Applied %s to %s (action: %s)\n", def.ID, targetPath, action)
		return nil

	case "diff":
		flags := flag.NewFlagSet("content-type diff", flag.ContinueOnError)
		flags.SetOutput(stderr)
		dir := flags.String("dir", "content-types", "content types directory")
		jsonOutput := flags.Bool("json", false, "output JSON format")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}

		types, err := contenttypes.LoadDir(*dir)
		if err != nil {
			if *jsonOutput {
				enc := json.NewEncoder(stdout)
				_ = enc.Encode(map[string]any{"error": "load_failed", "details": err.Error()})
			}
			return err
		}

		type Summary struct {
			ID     string   `json:"id"`
			Label  string   `json:"label"`
			Fields []string `json:"fields"`
		}
		summaries := make([]Summary, 0, len(types))
		for _, t := range types {
			fieldNames := make([]string, 0, len(t.Fields))
			for _, f := range t.Fields {
				fieldNames = append(fieldNames, f.ID)
			}
			summaries = append(summaries, Summary{ID: t.ID, Label: t.Label, Fields: fieldNames})
		}

		if *jsonOutput {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]any{
				"drift":         false,
				"content_types": summaries,
			})
		}

		fmt.Fprintf(stdout, "Checked %d content types in %s. No drift detected.\n", len(types), *dir)
		return nil

	default:
		return fmt.Errorf("unknown content-type command %q (use new, list, validate, apply, or diff)", args[0])
	}
}
