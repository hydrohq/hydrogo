package sdkdist

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// SDKLanguage selects generated client artifacts.
type SDKLanguage string

const (
	SDKTypeScript SDKLanguage = "typescript"
	SDKDart       SDKLanguage = "dart"
	SDKAll        SDKLanguage = "all"
)

const maxOpenAPIDocumentBytes = 32 << 20

type openAPIDocument struct {
	OpenAPI string `json:"openapi"`
	Info    struct {
		Title   string `json:"title"`
		Version string `json:"version"`
	} `json:"info"`
	Paths      map[string]openAPIPathItem `json:"paths"`
	Components openAPIComponents          `json:"components"`
}

type openAPIComponents struct {
	Schemas       map[string]json.RawMessage    `json:"schemas"`
	Responses     map[string]openAPIResponse    `json:"responses"`
	RequestBodies map[string]openAPIRequestBody `json:"requestBodies"`
}

type openAPIMediaType struct {
	Schema json.RawMessage `json:"schema"`
}

type openAPIRequestBody struct {
	Ref      string                      `json:"$ref"`
	Required bool                        `json:"required"`
	Content  map[string]openAPIMediaType `json:"content"`
}

type openAPIResponse struct {
	Ref     string                      `json:"$ref"`
	Content map[string]openAPIMediaType `json:"content"`
}

type openAPIOperation struct {
	OperationID string                     `json:"operationId"`
	Parameters  []openAPIParameter         `json:"parameters"`
	RequestBody openAPIRequestBody         `json:"requestBody"`
	Responses   map[string]openAPIResponse `json:"responses"`
}

type openAPIParameter struct {
	Name     string          `json:"name"`
	In       string          `json:"in"`
	Required bool            `json:"required"`
	Schema   json.RawMessage `json:"schema"`
}
type openAPIPathItem struct {
	Parameters []openAPIParameter `json:"parameters"`
	Get        openAPIOperation   `json:"get"`
	Post       openAPIOperation   `json:"post"`
	Put        openAPIOperation   `json:"put"`
	Patch      openAPIOperation   `json:"patch"`
	Delete     openAPIOperation   `json:"delete"`
	Head       openAPIOperation   `json:"head"`
	Options    openAPIOperation   `json:"options"`
	Trace      openAPIOperation   `json:"trace"`
}

type sdkOperation struct {
	Name         string
	Method       string
	Path         string
	Parameters   []sdkParameter
	BodySchema   json.RawMessage
	BodyRequired bool
	ReturnSchema json.RawMessage
	NoContent    bool
}

type sdkParameter struct {
	Name     string
	WireName string
	In       string
	Required bool
	Schema   json.RawMessage
}

var sdkIdentifier = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// GenerateOpenAPISDK reads an OpenAPI 3.1 JSON document and writes dependency-free
// TypeScript and/or Dart clients. The destination is created if it does not exist.
func GenerateOpenAPISDK(specPath, out string, language SDKLanguage) ([]string, error) {
	if language != SDKAll && language != SDKTypeScript && language != SDKDart {
		return nil, fmt.Errorf("unsupported SDK language %q", language)
	}
	if strings.TrimSpace(specPath) == "" || strings.TrimSpace(out) == "" {
		return nil, errors.New("OpenAPI input and output directory are required")
	}
	file, err := os.Open(specPath)
	if err != nil {
		return nil, fmt.Errorf("read OpenAPI document: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat OpenAPI document: %w", err)
	}
	if info.Size() <= 0 || info.Size() > maxOpenAPIDocumentBytes {
		return nil, fmt.Errorf("OpenAPI document must be between 1 and %d bytes", maxOpenAPIDocumentBytes)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxOpenAPIDocumentBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read OpenAPI document: %w", err)
	}
	if len(data) == 0 || len(data) > maxOpenAPIDocumentBytes {
		return nil, fmt.Errorf("OpenAPI document must be between 1 and %d bytes", maxOpenAPIDocumentBytes)
	}
	var doc openAPIDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse OpenAPI document: %w", err)
	}
	if err := validateOpenAPIDocument(data, doc); err != nil {
		return nil, err
	}
	operations, err := sdkOperations(doc)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	if language == SDKAll || language == SDKTypeScript {
		body, err := renderTypeScript(doc, operations)
		if err != nil {
			return nil, err
		}
		files["typescript/client.ts"] = body
	}
	if language == SDKAll || language == SDKDart {
		body, err := renderDart(doc, operations)
		if err != nil {
			return nil, err
		}
		files["dart/lib/hydro_client.dart"] = body
	}
	if language == SDKAll || language == SDKTypeScript {
		packageJSON, _ := json.MarshalIndent(struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Type    string `json:"type"`
			Exports string `json:"exports"`
		}{Name: "hydro-sdk-client", Version: sdkPackageVersion(doc.Info.Version), Type: "module", Exports: "./client.ts"}, "", "  ")
		files["typescript/package.json"] = append(packageJSON, '\n')
		files["typescript/tsconfig.json"] = []byte("{\n  \"compilerOptions\": {\n    \"target\": \"ES2022\",\n    \"module\": \"NodeNext\",\n    \"moduleResolution\": \"NodeNext\",\n    \"strict\": true,\n    \"exactOptionalPropertyTypes\": true,\n    \"noUncheckedIndexedAccess\": true,\n    \"noEmit\": true,\n    \"lib\": [\"ES2022\", \"DOM\"]\n  },\n  \"include\": [\"client.ts\"]\n}\n")
	}
	if language == SDKAll || language == SDKDart {
		files["dart/pubspec.yaml"] = []byte("name: hydro_sdk_client\nversion: " + sdkPackageVersion(doc.Info.Version) + "\nenvironment:\n  sdk: '>=3.2.0 <4.0.0'\n\n# Uses dart:io and the Dart standard library; no package dependencies.\n")
	}
	files["README.md"] = []byte("# HydroGo generated SDKs\n\nClients generated from an OpenAPI 3.1 JSON document. TypeScript uses the Fetch API; Dart includes a standard-library `dart:io` transport for Flutter native and Dart server applications. Both allow a custom transport.\n\n- TypeScript: `typescript/client.ts`\n- Dart: `dart/lib/hydro_client.dart`\n- Source contract: supplied with `--openapi`\n\nGenerated files are deterministic for a given contract and generator version.\n")
	if err := os.MkdirAll(out, 0o755); err != nil {
		return nil, err
	}
	written := make([]string, 0, len(files))
	for name, body := range files {
		path := filepath.Join(out, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := writeSDKFile(path, body); err != nil {
			return nil, err
		}
		written = append(written, path)
	}
	sort.Strings(written)
	return written, nil
}

var sdkSemver = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)

func sdkPackageVersion(value string) string {
	if sdkSemver.MatchString(value) {
		return value
	}
	return "0.0.0"
}

func writeSDKFile(path string, body []byte) error {
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return err
	}
	return nil
}

// Ensure the template output remains valid UTF-8 source and that a destination
// path cannot escape the requested output directory.
func validSDKPath(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
