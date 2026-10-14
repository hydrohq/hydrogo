package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var resourceFieldPart = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)

// interactiveResourceCommand authors the same Blueprint consumed by scaffold;
// it is intentionally a thin prompt layer, not a second schema language.
func interactiveResourceCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	out, name := ".", ""
	var fieldParts []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		key, value, hasValue := strings.Cut(arg, "=")
		switch key {
		case "--out", "--fields", "--field":
			if !hasValue {
				i++
				if i == len(args) {
					return fmt.Errorf("%s requires a value", key)
				}
				value = args[i]
			}
			switch key {
			case "--out":
				out = value
			case "--fields", "--field":
				fieldParts = append(fieldParts, value)
			}
		default:
			if strings.HasPrefix(arg, "-") || name != "" {
				return errors.New("usage: hydro new resource [name] [--fields name:type,...] [--out DIR]")
			}
			name = arg
		}
	}
	in := bufio.NewReader(stdin)
	read := func(prompt string) (string, error) {
		if _, err := fmt.Fprint(stdout, prompt); err != nil {
			return "", err
		}
		value, err := in.ReadString('\n')
		return strings.TrimSpace(value), err
	}
	if name == "" {
		value, err := read("Resource name: ")
		if err != nil {
			return err
		}
		name = value
	}
	fields := strings.Join(fieldParts, ",")
	if fields == "" {
		value, err := read("Fields (name:type[:required][:unique], comma-separated; relation: name:relation:collection[:one|many]): ")
		if err != nil {
			return err
		}
		fields = value
	}
	if !resourceFieldPart.MatchString(strings.ToLower(name)) || fields == "" {
		return errors.New("resource name and fields are required")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "name: %s\nfields:\n", name)
	titleField := ""
	for _, raw := range strings.Split(fields, ",") {
		parts := strings.Split(strings.TrimSpace(raw), ":")
		if len(parts) < 2 || len(parts) > 4 || !resourceFieldPart.MatchString(parts[0]) || !resourceFieldPart.MatchString(strings.ToLower(parts[1])) {
			return fmt.Errorf("invalid field %q", raw)
		}
		fmt.Fprintf(&b, "  - name: %s\n    type: %s\n", parts[0], parts[1])
		if titleField == "" && (strings.EqualFold(parts[1], "string") || strings.EqualFold(parts[1], "text")) {
			titleField = parts[0]
			b.WriteString("    ui:\n      list: true\n      searchable: true\n")
		}
		if strings.EqualFold(parts[1], "relation") || strings.EqualFold(parts[1], "reference") {
			if len(parts) < 3 || !resourceFieldPart.MatchString(parts[2]) {
				return fmt.Errorf("relation field %q requires a target collection", parts[0])
			}
			cardinality := "one"
			if len(parts) == 4 {
				cardinality = strings.ToLower(parts[3])
			}
			if cardinality != "one" && cardinality != "many" {
				return fmt.Errorf("relation field %q cardinality must be one or many", parts[0])
			}
			fmt.Fprintf(&b, "    relation:\n      collection: %s\n      cardinality: %s\n", parts[2], cardinality)
			if cardinality == "many" {
				b.WriteString("      max_items: 50\n")
			}
		} else {
			required, unique := false, false
			for _, option := range parts[2:] {
				switch strings.ToLower(option) {
				case "required":
					required = true
				case "unique":
					unique = true
				default:
					return fmt.Errorf("unsupported field option %q", option)
				}
			}
			if required {
				b.WriteString("    required: true\n")
			}
			if unique {
				b.WriteString("    validation:\n      unique: true\n")
			}
		}
	}
	if titleField != "" {
		fmt.Fprintf(&b, "admin:\n  use_as_title: %s\n", titleField)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	path := filepath.Join(out, strings.ToLower(name)+".blueprint.yaml")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("blueprint already exists: %s", path)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return err
	}
	_, err := fmt.Fprintf(stdout, "blueprint-created path=%s\n", path)
	if err != nil {
		return err
	}
	// Re-enter the canonical scaffold pipeline; interactive mode never grows a
	// second generator or bypasses the generated/custom boundary.
	return generateScaffoldCommand([]string{"--blueprint", path, "--out", out}, stdout, stderr)
}
