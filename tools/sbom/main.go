// Command sbom writes a deterministic SPDX 2.3 document for the Go module graph.
// It intentionally uses only the Go toolchain and standard library.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type module struct {
	Path    string
	Version string
	Sum     string
	Main    bool
}

type spdxDocument struct {
	SPDXVersion       string         `json:"spdxVersion"`
	DataLicense       string         `json:"dataLicense"`
	SPDXID            string         `json:"SPDXID"`
	Name              string         `json:"name"`
	DocumentNamespace string         `json:"documentNamespace"`
	CreationInfo      creationInfo   `json:"creationInfo"`
	Packages          []spdxPackage  `json:"packages"`
	Relationships     []relationship `json:"relationships"`
}

type creationInfo struct {
	Created  string   `json:"created"`
	Creators []string `json:"creators"`
}

type spdxPackage struct {
	SPDXID                  string        `json:"SPDXID"`
	Name                    string        `json:"name"`
	VersionInfo             string        `json:"versionInfo,omitempty"`
	DownloadLocation        string        `json:"downloadLocation"`
	FilesAnalyzed           bool          `json:"filesAnalyzed"`
	LicenseConcluded        string        `json:"licenseConcluded"`
	LicenseDeclared         string        `json:"licenseDeclared"`
	CopyrightText           string        `json:"copyrightText"`
	PackageVerificationCode string        `json:"packageVerificationCode,omitempty"`
	Checksums               []checksum    `json:"checksums,omitempty"`
	ExternalRefs            []externalRef `json:"externalRefs,omitempty"`
}

type checksum struct {
	Algorithm     string `json:"algorithm"`
	ChecksumValue string `json:"checksumValue"`
}

type externalRef struct {
	ReferenceCategory string `json:"referenceCategory"`
	ReferenceType     string `json:"referenceType"`
	ReferenceLocator  string `json:"referenceLocator"`
}

type relationship struct {
	SPDXElementID      string `json:"spdxElementId"`
	RelationshipType   string `json:"relationshipType"`
	RelatedSPDXElement string `json:"relatedSpdxElement"`
}

func main() {
	output := flag.String("output", "sbom.spdx.json", "output SPDX JSON path")
	flag.Parse()

	mods, err := moduleGraph()
	if err != nil {
		fail(err)
	}
	doc := buildDocument(mods)
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fail(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(*output, data, 0o600); err != nil {
		fail(fmt.Errorf("write %s: %w", *output, err))
	}
}

func moduleGraph() ([]module, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list -m -json all: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	dec := json.NewDecoder(&stdout)
	var modules []module
	for {
		var mod module
		err := dec.Decode(&mod)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode module graph: %w", err)
		}
		modules = append(modules, mod)
	}
	sort.Slice(modules, func(i, j int) bool {
		if modules[i].Path == modules[j].Path {
			return modules[i].Version < modules[j].Version
		}
		return modules[i].Path < modules[j].Path
	})
	return modules, nil
}

func buildDocument(modules []module) spdxDocument {
	created := "1970-01-01T00:00:00Z"
	if raw := os.Getenv("SOURCE_DATE_EPOCH"); raw != "" {
		if unix, err := strconv.ParseInt(raw, 10, 64); err == nil {
			created = time.Unix(unix, 0).UTC().Format(time.RFC3339)
		}
	}
	doc := spdxDocument{
		SPDXVersion:       "SPDX-2.3",
		DataLicense:       "CC0-1.0",
		SPDXID:            "SPDXRef-DOCUMENT",
		Name:              "HydroGo Go module dependencies",
		DocumentNamespace: "https://hydrogo.dev/sbom/go-module-graph",
		CreationInfo:      creationInfo{Created: created, Creators: []string{"Tool: hydrogo-sbom"}},
		Packages:          make([]spdxPackage, 0, len(modules)),
		Relationships:     make([]relationship, 0, len(modules)),
	}
	for i, mod := range modules {
		id := fmt.Sprintf("SPDXRef-Package-%d", i+1)
		pkg := spdxPackage{
			SPDXID:           id,
			Name:             mod.Path,
			VersionInfo:      mod.Version,
			DownloadLocation: "NOASSERTION",
			FilesAnalyzed:    false,
			LicenseConcluded: "NOASSERTION",
			LicenseDeclared:  "NOASSERTION",
			CopyrightText:    "NOASSERTION",
		}
		// Go's h1 checksum is not an SPDX SHA256 checksum. Keep it out of the
		// SPDX checksum field rather than emitting a misleading algorithm label.
		if mod.Version != "" {
			pkg.ExternalRefs = []externalRef{{ReferenceCategory: "PACKAGE-MANAGER", ReferenceType: "purl", ReferenceLocator: "pkg:golang/" + mod.Path + "@" + mod.Version}}
		}
		doc.Packages = append(doc.Packages, pkg)
		doc.Relationships = append(doc.Relationships, relationship{SPDXElementID: "SPDXRef-DOCUMENT", RelationshipType: "DESCRIBES", RelatedSPDXElement: id})
	}
	return doc
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "hydrogo-sbom:", err)
	os.Exit(1)
}
