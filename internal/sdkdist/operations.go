package sdkdist

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

func sdkOperations(doc openAPIDocument) ([]sdkOperation, error) {
	var result []sdkOperation
	for path, pathItem := range doc.Paths {
		methods := map[string]openAPIOperation{"get": pathItem.Get, "post": pathItem.Post, "put": pathItem.Put, "patch": pathItem.Patch, "delete": pathItem.Delete, "head": pathItem.Head, "options": pathItem.Options, "trace": pathItem.Trace}
		for method, operation := range methods {
			if operation.OperationID == "" && len(operation.Parameters) == 0 && len(operation.Responses) == 0 && len(operation.RequestBody.Content) == 0 {
				continue
			}
			name := operation.OperationID
			if name == "" {
				name = method + path
			}
			name = sdkIdentifier.ReplaceAllString(name, "_")
			name = safeIdentifier(name, "request_")
			allParams := append(append([]openAPIParameter(nil), pathItem.Parameters...), operation.Parameters...)
			params := make([]sdkParameter, 0, len(allParams))
			for _, p := range allParams {
				if p.In == "query" || p.In == "path" {
					params = append(params, sdkParameter{Name: safeIdentifier(p.Name, "arg_"), WireName: p.Name, In: p.In, Required: p.Required || p.In == "path", Schema: p.Schema})
				}
			}
			sort.Slice(params, func(i, j int) bool {
				if params[i].Required != params[j].Required {
					return params[i].Required
				}
				return params[i].Name < params[j].Name
			})
			item := sdkOperation{Name: name, Method: strings.ToUpper(method), Path: path, Parameters: params, BodyRequired: operation.RequestBody.Required}
			requestBody, err := resolveRequestBody(operation.RequestBody, doc.Components.RequestBodies)
			if err != nil {
				return nil, err
			}
			item.BodyRequired = requestBody.Required
			item.BodySchema = mediaSchema(requestBody.Content)
			responses := make(map[string]openAPIResponse, len(operation.Responses))
			for code, response := range operation.Responses {
				resolved, resolveErr := resolveResponse(response, doc.Components.Responses)
				if resolveErr != nil {
					return nil, resolveErr
				}
				responses[code] = resolved
			}
			item.ReturnSchema = successSchema(responses)
			item.NoContent = operationHasOnlyNoContentResponses(responses)
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		return result[i].Method < result[j].Method
	})
	return result, nil
}

func resolveRequestBody(body openAPIRequestBody, components map[string]openAPIRequestBody) (openAPIRequestBody, error) {
	for depth := 0; body.Ref != ""; depth++ {
		if depth >= 16 {
			return openAPIRequestBody{}, errors.New("OpenAPI request body reference depth exceeded")
		}
		name, ok := componentReferenceName(body.Ref, "requestBodies")
		if !ok {
			return openAPIRequestBody{}, fmt.Errorf("unsupported OpenAPI request body reference %q", body.Ref)
		}
		resolved, ok := components[name]
		if !ok {
			return openAPIRequestBody{}, fmt.Errorf("unresolved OpenAPI request body reference %q", body.Ref)
		}
		body = resolved
	}
	return body, nil
}

func resolveResponse(response openAPIResponse, components map[string]openAPIResponse) (openAPIResponse, error) {
	for depth := 0; response.Ref != ""; depth++ {
		if depth >= 16 {
			return openAPIResponse{}, errors.New("OpenAPI response reference depth exceeded")
		}
		name, ok := componentReferenceName(response.Ref, "responses")
		if !ok {
			return openAPIResponse{}, fmt.Errorf("unsupported OpenAPI response reference %q", response.Ref)
		}
		resolved, ok := components[name]
		if !ok {
			return openAPIResponse{}, fmt.Errorf("unresolved OpenAPI response reference %q", response.Ref)
		}
		response = resolved
	}
	return response, nil
}

func componentReferenceName(reference, componentType string) (string, bool) {
	prefix := "#/components/" + componentType + "/"
	if !strings.HasPrefix(reference, prefix) {
		return "", false
	}
	name, err := url.PathUnescape(strings.TrimPrefix(reference, prefix))
	if err != nil || name == "" || strings.Contains(name, "/") {
		return "", false
	}
	name = strings.ReplaceAll(strings.ReplaceAll(name, "~1", "/"), "~0", "~")
	return name, true
}

func operationHasOnlyNoContentResponses(responses map[string]openAPIResponse) bool {
	hasSuccess := false
	for code, response := range responses {
		if len(code) != 3 || code[0] != '2' {
			continue
		}
		hasSuccess = true
		if len(response.Content) > 0 {
			return false
		}
	}
	return hasSuccess
}

func safeIdentifier(value, fallback string) string {
	value = sdkIdentifier.ReplaceAllString(strings.TrimSpace(value), "_")
	if value == "" || !(value[0] >= 'a' && value[0] <= 'z' || value[0] >= 'A' && value[0] <= 'Z' || value[0] == '_') {
		value = fallback + value
	}
	switch value {
	case "class", "default", "function", "new", "return", "var", "const", "let", "await", "yield", "enum", "extends", "implements", "interface", "package", "private", "protected", "public", "static", "super", "this", "throw", "try", "catch", "finally", "switch", "case", "break", "continue", "for", "while", "if", "else", "import", "export", "with", "delete", "typeof", "void", "in", "instanceof", "as", "async", "required", "String", "int", "double", "bool", "Object", "dynamic", "final", "true", "false", "null":
		value = "arg_" + value
	}
	return value
}

func mediaSchema(content map[string]openAPIMediaType) json.RawMessage {
	if item, ok := content["application/json"]; ok {
		return item.Schema
	}
	keys := make([]string, 0, len(content))
	for key := range content {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > 0 {
		return content[keys[0]].Schema
	}
	return nil
}

func successSchema(responses map[string]openAPIResponse) json.RawMessage {
	codes := make([]string, 0, len(responses))
	for code := range responses {
		if len(code) == 3 && code[0] == '2' {
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)
	for _, code := range codes {
		if raw := mediaSchema(responses[code].Content); len(raw) > 0 {
			return raw
		}
	}
	return nil
}
