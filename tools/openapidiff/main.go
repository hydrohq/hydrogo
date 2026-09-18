package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

// OpenAPISpec represents the minimal structural contract required to detect breaking changes.
type OpenAPISpec struct {
	OpenAPI string                          `json:"openapi"`
	Paths   map[string]map[string]Operation `json:"paths"`
}

type Operation struct {
	OperationID string              `json:"operationId"`
	Parameters  []Parameter         `json:"parameters"`
	Responses   map[string]Response `json:"responses"`
}

type Parameter struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
}

type Response struct {
	Description string `json:"description"`
}

func DiffOpenAPISpecs(baselineData, currentData []byte) ([]string, error) {
	var baseline, current OpenAPISpec
	if err := json.Unmarshal(baselineData, &baseline); err != nil {
		return nil, fmt.Errorf("failed to parse baseline spec: %w", err)
	}
	if err := json.Unmarshal(currentData, &current); err != nil {
		return nil, fmt.Errorf("failed to parse current spec: %w", err)
	}

	var breakingChanges []string

	// Check for removed paths or methods
	for path, baselineMethods := range baseline.Paths {
		currentMethods, pathExists := current.Paths[path]
		if !pathExists {
			breakingChanges = append(breakingChanges, fmt.Sprintf("Removed endpoint path: %s", path))
			continue
		}

		for method, baselineOp := range baselineMethods {
			currentOp, methodExists := currentMethods[method]
			if !methodExists {
				breakingChanges = append(breakingChanges, fmt.Sprintf("Removed HTTP method %s on path %s", method, path))
				continue
			}

			// Check for removed responses
			for statusCode := range baselineOp.Responses {
				if _, respExists := currentOp.Responses[statusCode]; !respExists {
					breakingChanges = append(breakingChanges, fmt.Sprintf("Removed response status code %s from %s %s", statusCode, method, path))
				}
			}

			// Check for newly required parameters in current
			baselineParamMap := make(map[string]Parameter)
			for _, p := range baselineOp.Parameters {
				baselineParamMap[p.Name] = p
			}

			for _, currParam := range currentOp.Parameters {
				baseParam, existed := baselineParamMap[currParam.Name]
				if (!existed && currParam.Required) || (existed && !baseParam.Required && currParam.Required) {
					breakingChanges = append(breakingChanges, fmt.Sprintf("Added required parameter '%s' to %s %s", currParam.Name, method, path))
				}
			}
		}
	}

	return breakingChanges, nil
}

func main() {
	baselinePath := flag.String("baseline", "", "Path to baseline OpenAPI JSON")
	currentPath := flag.String("current", "", "Path to current OpenAPI JSON")
	flag.Parse()

	if *baselinePath == "" || *currentPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: openapidiff --baseline <baseline.json> --current <current.json>")
		os.Exit(2)
	}

	baselineBytes, err := os.ReadFile(*baselinePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading baseline file: %v\n", err)
		os.Exit(1)
	}

	currentBytes, err := os.ReadFile(*currentPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading current file: %v\n", err)
		os.Exit(1)
	}

	breaking, err := DiffOpenAPISpecs(baselineBytes, currentBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error comparing specs: %v\n", err)
		os.Exit(1)
	}

	if len(breaking) > 0 {
		fmt.Fprintln(os.Stderr, "BREAKING API CHANGES DETECTED:")
		for _, change := range breaking {
			fmt.Fprintf(os.Stderr, " - %s\n", change)
		}
		os.Exit(1)
	}

	fmt.Println("PASS: OpenAPI contract is backwards-compatible.")
}

var ErrBreakingChange = errors.New("breaking API change detected")
