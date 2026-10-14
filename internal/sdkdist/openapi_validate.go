package sdkdist

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func validateOpenAPIDocument(raw []byte, doc openAPIDocument) error {
	if !strings.HasPrefix(doc.OpenAPI, "3.1.") || strings.TrimSpace(doc.Info.Title) == "" || strings.TrimSpace(doc.Info.Version) == "" || len(doc.Paths) == 0 {
		return errors.New("an OpenAPI 3.1 document with info and at least one path is required")
	}
	for path := range doc.Paths {
		if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#") {
			return fmt.Errorf("invalid OpenAPI path %q", path)
		}
	}
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return fmt.Errorf("parse OpenAPI document: %w", err)
	}
	rootObject, _ := root.(map[string]any)
	pathItems, _ := rootObject["paths"].(map[string]any)
	for path, rawItem := range pathItems {
		item, _ := rawItem.(map[string]any)
		for _, method := range []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"} {
			rawOperation, exists := item[method]
			if !exists {
				continue
			}
			operation, _ := rawOperation.(map[string]any)
			if responses, ok := operation["responses"].(map[string]any); !ok || len(responses) == 0 {
				return fmt.Errorf("OpenAPI operation %s %s must declare responses", strings.ToUpper(method), path)
			}
		}
	}
	var validateRefs func(any) error
	validateRefs = func(value any) error {
		switch value := value.(type) {
		case map[string]any:
			if ref, ok := value["$ref"].(string); ok {
				if err := validateOpenAPIRef(root, ref); err != nil {
					return err
				}
			}
			for _, child := range value {
				if err := validateRefs(child); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range value {
				if err := validateRefs(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return validateRefs(root)
}

func validateOpenAPIRef(root any, reference string) error {
	if !strings.HasPrefix(reference, "#/") {
		return fmt.Errorf("unsupported external OpenAPI reference %q", reference)
	}
	fragment, err := url.PathUnescape(strings.TrimPrefix(reference, "#"))
	if err != nil {
		return fmt.Errorf("invalid OpenAPI reference %q", reference)
	}
	parts := strings.Split(strings.TrimPrefix(fragment, "/"), "/")
	if len(parts) < 3 || parts[0] != "components" {
		return fmt.Errorf("unsupported OpenAPI reference %q", reference)
	}
	for index, part := range parts {
		parts[index] = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
	}
	if _, ok := lookupJSONPointer(root, parts); !ok {
		return fmt.Errorf("unresolved OpenAPI reference %q", reference)
	}
	return nil
}

func lookupJSONPointer(root any, parts []string) (any, bool) {
	value := root
	for _, part := range parts {
		switch current := value.(type) {
		case map[string]any:
			next, ok := current[part]
			if !ok {
				return nil, false
			}
			value = next
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(current) {
				return nil, false
			}
			value = current[index]
		default:
			return nil, false
		}
	}
	return value, true
}
