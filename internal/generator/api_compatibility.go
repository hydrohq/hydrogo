package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// APICompatibilityReport describes a non-breaking subset check between two
// OpenAPI documents. It intentionally checks only published paths/methods and
// response schema properties; semantic business compatibility remains a
// reviewed module concern.
type APICompatibilityReport struct {
	PathsChecked      int
	MethodsChecked    int
	PropertiesChecked int
	ParametersChecked int
}

func CheckAPICompatibility(baselinePath, currentPath string) (APICompatibilityReport, error) {
	baseline, err := loadOpenAPI(baselinePath)
	if err != nil {
		return APICompatibilityReport{}, fmt.Errorf("baseline OpenAPI: %w", err)
	}
	current, err := loadOpenAPI(currentPath)
	if err != nil {
		return APICompatibilityReport{}, fmt.Errorf("current OpenAPI: %w", err)
	}
	oldPaths, _ := baseline["paths"].(map[string]any)
	newPaths, _ := current["paths"].(map[string]any)
	if len(oldPaths) == 0 {
		return APICompatibilityReport{}, fmt.Errorf("baseline OpenAPI has no paths")
	}
	report := APICompatibilityReport{}
	for path, raw := range oldPaths {
		newRaw, ok := newPaths[path]
		if !ok {
			return report, fmt.Errorf("removed API path %s", path)
		}
		report.PathsChecked++
		oldMethods, _ := raw.(map[string]any)
		newMethods, _ := newRaw.(map[string]any)
		for method, oldOperation := range oldMethods {
			if method == "parameters" || method == "summary" || method == "description" {
				continue
			}
			newOperation, ok := newMethods[method]
			if !ok {
				return report, fmt.Errorf("removed API operation %s %s", method, path)
			}
			report.MethodsChecked++
			oldOp, _ := oldOperation.(map[string]any)
			newOp, _ := newOperation.(map[string]any)
			if err := compareSchemas(path, method, oldOp, newOp, baseline, current, &report); err != nil {
				return report, err
			}
			if err := compareRequestContract(path, method, oldOp, newOp, &report); err != nil {
				return report, err
			}
		}
	}
	return report, nil
}

func compareRequestContract(path, method string, oldOp, newOp map[string]any, report *APICompatibilityReport) error {
	oldParameters, _ := oldOp["parameters"].([]any)
	newParameters, _ := newOp["parameters"].([]any)
	newByKey := make(map[string]map[string]any, len(newParameters))
	for _, raw := range newParameters {
		parameter, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid current parameter for %s %s", method, path)
		}
		key := fmt.Sprintf("%v:%v", parameter["in"], parameter["name"])
		newByKey[key] = parameter
	}
	for _, raw := range oldParameters {
		parameter, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid baseline parameter for %s %s", method, path)
		}
		key := fmt.Sprintf("%v:%v", parameter["in"], parameter["name"])
		current, ok := newByKey[key]
		if !ok {
			return fmt.Errorf("removed request parameter %s for %s %s", key, method, path)
		}
		if oldRequired, _ := parameter["required"].(bool); !oldRequired {
			if currentRequired, _ := current["required"].(bool); currentRequired {
				return fmt.Errorf("request parameter %s became required for %s %s", key, method, path)
			}
		}
		report.ParametersChecked++
	}
	if _, oldBody := oldOp["requestBody"]; oldBody {
		if _, newBody := newOp["requestBody"]; !newBody {
			return fmt.Errorf("removed request body for %s %s", method, path)
		}
	}
	return nil
}

func loadOpenAPI(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	if document["openapi"] == nil {
		return nil, fmt.Errorf("missing openapi version")
	}
	return document, nil
}

func compareSchemas(path, method string, oldOp, newOp, baseline, current map[string]any, report *APICompatibilityReport) error {
	oldResponses, _ := oldOp["responses"].(map[string]any)
	newResponses, _ := newOp["responses"].(map[string]any)
	for code, oldResponse := range oldResponses {
		newResponse, ok := newResponses[code]
		if !ok {
			return fmt.Errorf("removed response %s for %s %s", code, method, path)
		}
		oldResponseMap, ok := oldResponse.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid baseline response %s for %s %s", code, method, path)
		}
		newResponseMap, ok := newResponse.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid current response %s for %s %s", code, method, path)
		}
		oldContent, _ := oldResponseMap["content"].(map[string]any)
		newContent, _ := newResponseMap["content"].(map[string]any)
		for media, oldMedia := range oldContent {
			newMedia, ok := newContent[media]
			if !ok {
				return fmt.Errorf("removed response media %s for %s %s", media, method, path)
			}
			oldMediaMap, ok := oldMedia.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid baseline media %s for %s %s", media, method, path)
			}
			newMediaMap, ok := newMedia.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid current media %s for %s %s", media, method, path)
			}
			oldSchema, _ := oldMediaMap["schema"].(map[string]any)
			newSchema, _ := newMediaMap["schema"].(map[string]any)
			oldSchema = resolveLocalSchema(baseline, oldSchema)
			newSchema = resolveLocalSchema(current, newSchema)
			oldProperties, _ := oldSchema["properties"].(map[string]any)
			newProperties, _ := newSchema["properties"].(map[string]any)
			for property := range oldProperties {
				if _, ok := newProperties[property]; !ok {
					return fmt.Errorf("removed response property %s for %s %s", property, method, path)
				}
				report.PropertiesChecked++
			}
		}
	}
	return nil
}

func resolveLocalSchema(document map[string]any, schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	ref, _ := schema["$ref"].(string)
	const schemaPrefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, schemaPrefix) {
		return schema
	}
	components, _ := document["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	resolved, _ := schemas[strings.TrimPrefix(ref, schemaPrefix)].(map[string]any)
	if resolved == nil {
		return schema
	}
	return resolveLocalSchema(document, resolved)
}
