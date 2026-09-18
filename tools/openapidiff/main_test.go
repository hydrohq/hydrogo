package main

import (
	"testing"
)

func TestDiffOpenAPISpecs_NoChanges(t *testing.T) {
	spec := []byte(`{
		"openapi": "3.0.3",
		"paths": {
			"/api/v1/pages": {
				"get": {
					"operationId": "listPages",
					"responses": {
						"200": { "description": "OK" }
					}
				}
			}
		}
	}`)

	breaking, err := DiffOpenAPISpecs(spec, spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(breaking) != 0 {
		t.Fatalf("expected 0 breaking changes, got %d: %v", len(breaking), breaking)
	}
}

func TestDiffOpenAPISpecs_BreakingChangesDetected(t *testing.T) {
	baseline := []byte(`{
		"openapi": "3.0.3",
		"paths": {
			"/api/v1/pages": {
				"get": {
					"operationId": "listPages",
					"parameters": [
						{ "name": "limit", "in": "query", "required": false }
					],
					"responses": {
						"200": { "description": "OK" },
						"404": { "description": "Not Found" }
					}
				},
				"post": {
					"operationId": "createPage",
					"responses": { "201": { "description": "Created" } }
				}
			}
		}
	}`)

	current := []byte(`{
		"openapi": "3.0.3",
		"paths": {
			"/api/v1/pages": {
				"get": {
					"operationId": "listPages",
					"parameters": [
						{ "name": "limit", "in": "query", "required": true }
					],
					"responses": {
						"200": { "description": "OK" }
					}
				}
			}
		}
	}`)

	breaking, err := DiffOpenAPISpecs(baseline, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(breaking) == 0 {
		t.Fatalf("expected breaking changes to be detected, got none")
	}

	expectedFound := 0
	for _, b := range breaking {
		if b == "Removed HTTP method post on path /api/v1/pages" ||
			b == "Removed response status code 404 from get /api/v1/pages" ||
			b == "Added required parameter 'limit' to get /api/v1/pages" {
			expectedFound++
		}
	}
	if expectedFound < 3 {
		t.Errorf("expected 3 specific breaking changes, found %d: %v", expectedFound, breaking)
	}
}
