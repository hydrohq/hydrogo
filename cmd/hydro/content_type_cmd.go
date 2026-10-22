package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ardaozyildirim/hydrogo/internal/contenttypes"
)

var contentTypeID = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// contentTypeCommand implements `hydro content-type new|validate`.
func contentTypeCommand(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: hydro content-type new ID [--dir content-types] | validate [--dir content-types]")
	}
	flags := flag.NewFlagSet("content-type "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", "content-types", "content types directory")
	switch args[0] {
	case "new":
		// Accept `new ID --dir X` as well as `new --dir X ID`.
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
		if err := os.MkdirAll(*dir, 0o755); err != nil { // #nosec G301 ,G703-- content-types directory of the project
			return err
		}
		path := filepath.Join(*dir, id+".yaml")
		if _, err := os.Stat(path); err == nil { // #nosec G703 -- id matches contentTypeID; the directory is the operator's flag
			return fmt.Errorf("%s already exists", path)
		}
		skeleton := fmt.Sprintf(`# yaml-language-server: $schema=https://raw.githubusercontent.com/ozyildirims/hydrogo/main/docs/schemas/content-type.schema.json
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
		if err := os.WriteFile(path, []byte(skeleton), 0o644); err != nil { // #nosec G306 ,G703-- content type definition: a project source file with no secrets
			return err
		}
		fmt.Fprintf(stdout, "Created %s. Restart hydro serve to see /admin/content/%s and /api/v1/content/%s.\n", path, id, id)
		return nil
	case "validate":
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		types, err := contenttypes.LoadDir(*dir)
		if err != nil {
			return err
		}
		for _, t := range types {
			fmt.Fprintf(stdout, "ok  %s (%d fields)\n", t.ID, len(t.Fields))
		}
		fmt.Fprintf(stdout, "%d content types valid in %s\n", len(types), *dir)
		return nil
	default:
		return fmt.Errorf("unknown content-type command %q (use new or validate)", args[0])
	}
}
