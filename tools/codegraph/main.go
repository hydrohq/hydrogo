// Command codegraph maps the Go packages of this repository: which layer each belongs to, what
// it imports and what imports it. It writes three files under docs/architecture:
//
//	CODEMAP.md       a readable map (layers, hubs, module edges, package index)
//	codegraph.json   the same data for tools and AI assistants
//	codegraph.html   a self-contained, interactive graph (open it in a browser)
//
// Usage:
//
//	go run ./tools/codegraph          # regenerate the three files
//	go run ./tools/codegraph -check   # exit 1 when the committed files are out of date
//
// The output is deterministic (no timestamps, no absolute paths), so -check is reliable in CI.
// Boundary rules between modules are enforced by tools/moduleguard; this tool only shows the shape.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const outDir = "docs/architecture"

type listedPackage struct {
	ImportPath string
	Dir        string
	Doc        string
	GoFiles    []string
	Imports    []string
}

// Node is one Go package of the repository.
type Node struct {
	ID         string   `json:"id"`
	Layer      string   `json:"layer"`
	Doc        string   `json:"doc,omitempty"`
	Files      int      `json:"files"`
	FanIn      int      `json:"fan_in"`
	FanOut     int      `json:"fan_out"`
	Imports    []string `json:"imports"`
	ImportedBy []string `json:"imported_by"`
}

// ExternalDep is a third-party module family and how many repository packages import it.
type ExternalDep struct {
	Path     string `json:"path"`
	Packages int    `json:"packages"`
}

// Graph is the whole map.
type Graph struct {
	Module   string        `json:"module"`
	Nodes    []Node        `json:"nodes"`
	External []ExternalDep `json:"external"`
}

// addOnModules are separately distributed modules that may be applied into a working tree.
// They are left out so the map is identical whether or not they are present.
var addOnModules = []string{"modules/payments", "modules/shipping", "modules/invoicing"}

func isAddOn(id string) bool {
	for _, m := range addOnModules {
		if id == m || strings.HasPrefix(id, m+"/") {
			return true
		}
	}
	return false
}

var layerPurpose = map[string]string{
	"cmd":       "Binary entry points: CLI commands, server composition, HTTP routes and admin APIs.",
	"internal":  "Cross-cutting capabilities: auth, database, outbox, jobs, mailer, audit, config and more.",
	"modules":   "Feature modules: pages, news, members, site, seo, commerce.",
	"extension": "Public extension SDK.",
	"tools":     "Repository guards and developer tooling.",
	"tests":     "Contract and end-to-end tests.",
	"examples":  "Runnable examples.",
}

func main() {
	check := flag.Bool("check", false, "fail when the committed files are out of date instead of writing them")
	flag.Parse()
	g, err := load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "codegraph:", err)
		os.Exit(1)
	}
	files, err := render(g)
	if err != nil {
		fmt.Fprintln(os.Stderr, "codegraph:", err)
		os.Exit(1)
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	stale := false
	for _, name := range names {
		path := filepath.Join(outDir, name)
		if *check {
			if have, err := os.ReadFile(path); err != nil || !bytes.Equal(have, files[name]) { // #nosec G304 -- path is joined from fixed outDir and generated file map key
				fmt.Fprintf(os.Stderr, "codegraph: %s is out of date; run `go run ./tools/codegraph`\n", path)
				stale = true
			}
			continue
		}
		if err := os.MkdirAll(outDir, 0o750); err != nil {
			fmt.Fprintln(os.Stderr, "codegraph:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(path, files[name], 0o600); err != nil {
			fmt.Fprintln(os.Stderr, "codegraph:", err)
			os.Exit(1)
		}
		fmt.Println("wrote", path)
	}
	if stale {
		os.Exit(1)
	}
}

func load() (Graph, error) {
	mod, err := exec.Command("go", "list", "-m").Output()
	if err != nil {
		return Graph{}, fmt.Errorf("go list -m: %w", err)
	}
	out, err := exec.Command("go", "list", "-json=ImportPath,Dir,Doc,GoFiles,Imports", "./...").Output()
	if err != nil {
		return Graph{}, fmt.Errorf("go list: %w", err)
	}
	var pkgs []listedPackage
	dec := json.NewDecoder(bytes.NewReader(out))
	for dec.More() {
		var p listedPackage
		if err := dec.Decode(&p); err != nil {
			return Graph{}, err
		}
		pkgs = append(pkgs, p)
	}
	return build(strings.TrimSpace(string(mod)), pkgs), nil
}

// build turns go list output into the graph.
func build(module string, pkgs []listedPackage) Graph {
	g := Graph{Module: module}
	byID := map[string]*Node{}
	ext := map[string]map[string]bool{}
	for _, p := range pkgs {
		if len(p.GoFiles) == 0 {
			continue
		}
		id := strings.TrimPrefix(p.ImportPath, module+"/")
		if id == p.ImportPath {
			id = "."
		}
		if isAddOn(id) {
			continue
		}
		n := &Node{ID: id, Layer: layerOf(id), Doc: firstSentence(p.Doc), Files: len(p.GoFiles)}
		for _, imp := range p.Imports {
			switch {
			case strings.HasPrefix(imp, module+"/"):
				n.Imports = append(n.Imports, strings.TrimPrefix(imp, module+"/"))
			case isThirdParty(imp):
				fam := family(imp)
				if ext[fam] == nil {
					ext[fam] = map[string]bool{}
				}
				ext[fam][id] = true
			}
		}
		sort.Strings(n.Imports)
		byID[id] = n
	}
	for _, n := range byID {
		kept := n.Imports[:0]
		for _, imp := range n.Imports {
			if other, ok := byID[imp]; ok {
				kept = append(kept, imp)
				other.ImportedBy = append(other.ImportedBy, n.ID)
			}
		}
		n.Imports = kept
	}
	for _, n := range byID {
		sort.Strings(n.ImportedBy)
		n.FanIn, n.FanOut = len(n.ImportedBy), len(n.Imports)
		if n.Imports == nil {
			n.Imports = []string{}
		}
		if n.ImportedBy == nil {
			n.ImportedBy = []string{}
		}
		g.Nodes = append(g.Nodes, *n)
	}
	sort.Slice(g.Nodes, func(i, j int) bool { return g.Nodes[i].ID < g.Nodes[j].ID })
	for fam, users := range ext {
		g.External = append(g.External, ExternalDep{Path: fam, Packages: len(users)})
	}
	sort.Slice(g.External, func(i, j int) bool {
		if g.External[i].Packages != g.External[j].Packages {
			return g.External[i].Packages > g.External[j].Packages
		}
		return g.External[i].Path < g.External[j].Path
	})
	return g
}

func layerOf(id string) string {
	first, _, _ := strings.Cut(id, "/")
	if id == "." {
		return "root"
	}
	switch first {
	case "cmd", "internal", "modules", "extension", "tools", "tests", "examples":
		return first
	}
	return "other"
}

func isThirdParty(imp string) bool {
	first, _, _ := strings.Cut(imp, "/")
	return strings.Contains(first, ".")
}

// family trims an import path to its module family (host/owner/repo), good enough for a count.
func family(imp string) string {
	parts := strings.Split(imp, "/")
	switch parts[0] {
	case "github.com", "gitlab.com", "golang.org", "google.golang.org", "go.opentelemetry.io":
		if len(parts) >= 3 {
			return strings.Join(parts[:3], "/")
		}
	}
	if len(parts) >= 2 {
		return strings.Join(parts[:2], "/")
	}
	return imp
}

func firstSentence(doc string) string {
	doc = strings.TrimSpace(strings.ReplaceAll(doc, "\n", " "))
	if i := strings.Index(doc, ". "); i >= 0 {
		doc = doc[:i+1]
	}
	if len(doc) > 160 {
		doc = doc[:157] + "..."
	}
	return doc
}

func render(g Graph) (map[string][]byte, error) {
	js, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return nil, err
	}
	js = append(js, '\n')
	compact, err := json.Marshal(g)
	if err != nil {
		return nil, err
	}
	data := strings.ReplaceAll(string(compact), "</", `<\/`)
	html := strings.Replace(htmlTemplate, "__DATA__", data, 1)
	return map[string][]byte{
		"CODEMAP.md":     []byte(markdown(g)),
		"codegraph.json": js,
		"codegraph.html": []byte(html),
	}, nil
}

func markdown(g Graph) string {
	var b strings.Builder
	layers := map[string][]Node{}
	for _, n := range g.Nodes {
		layers[n.Layer] = append(layers[n.Layer], n)
	}
	names := make([]string, 0, len(layers))
	for l := range layers {
		names = append(names, l)
	}
	sort.Strings(names)

	b.WriteString("# Code map\n\n")
	b.WriteString("Generated by `go run ./tools/codegraph`; do not edit. CI fails when it is stale (`-check`).\n")
	b.WriteString("Open [codegraph.html](codegraph.html) in a browser for the interactive graph; [codegraph.json](codegraph.json) holds the same data.\n\n")
	b.WriteString("Read this first when you are new to the code or an AI assistant asked to change it: it shows where things live and which direction imports may flow. ")
	b.WriteString("Module-to-module rules are enforced by `tools/moduleguard`; see `AGENTS.md` for the rules themselves.\n\n")

	b.WriteString("## Layers\n\n| Layer | Packages | Purpose |\n|---|---:|---|\n")
	for _, l := range names {
		fmt.Fprintf(&b, "| `%s` | %d | %s |\n", l, len(layers[l]), layerPurpose[l])
	}

	edges := map[[2]string]int{}
	byID := map[string]Node{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	for _, n := range g.Nodes {
		for _, imp := range n.Imports {
			if to := byID[imp].Layer; to != n.Layer {
				edges[[2]string{n.Layer, to}]++
			}
		}
	}
	keys := make([][2]string, 0, len(edges))
	for k := range edges {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	b.WriteString("\n## How layers import each other\n\nThe number is how many package-to-package imports cross that line.\n\n```mermaid\nflowchart LR\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "  %s -->|%d| %s\n", k[0], edges[k], k[1])
	}
	b.WriteString("```\n")

	hubs := append([]Node(nil), g.Nodes...)
	sort.SliceStable(hubs, func(i, j int) bool { return hubs[i].FanIn > hubs[j].FanIn })
	b.WriteString("\n## Most imported packages\n\nChanging one of these touches the most code.\n\n| Package | Imported by | Imports |\n|---|---:|---:|\n")
	for _, n := range hubs[:min(15, len(hubs))] {
		fmt.Fprintf(&b, "| `%s` | %d | %d |\n", n.ID, n.FanIn, n.FanOut)
	}
	wide := append([]Node(nil), g.Nodes...)
	sort.SliceStable(wide, func(i, j int) bool { return wide[i].FanOut > wide[j].FanOut })
	b.WriteString("\n## Packages that import the most\n\nUsually composition code: the place to look when wiring changes.\n\n| Package | Imports | Imported by |\n|---|---:|---:|\n")
	for _, n := range wide[:min(10, len(wide))] {
		fmt.Fprintf(&b, "| `%s` | %d | %d |\n", n.ID, n.FanOut, n.FanIn)
	}

	b.WriteString("\n## Module dependencies\n\nPackages under `modules/` that import another module's packages.\n\n")
	any := false
	for _, n := range g.Nodes {
		if n.Layer != "modules" {
			continue
		}
		own := moduleOf(n.ID)
		deps := map[string]bool{}
		for _, imp := range n.Imports {
			if strings.HasPrefix(imp, "modules/") && moduleOf(imp) != own {
				deps[moduleOf(imp)] = true
			}
		}
		if len(deps) > 0 {
			any = true
			list := make([]string, 0, len(deps))
			for d := range deps {
				list = append(list, "`"+d+"`")
			}
			sort.Strings(list)
			fmt.Fprintf(&b, "- `%s` → %s\n", n.ID, strings.Join(list, ", "))
		}
	}
	if !any {
		b.WriteString("None.\n")
	}

	b.WriteString("\n## Package index\n\nEach line: package, one-sentence purpose (its package comment), imported by / imports.\n")
	for _, l := range names {
		fmt.Fprintf(&b, "\n### %s\n\n", l)
		for _, n := range layers[l] {
			doc := n.Doc
			if doc == "" {
				doc = "(no package comment)"
			}
			fmt.Fprintf(&b, "- `%s`: %s (in %d, out %d)\n", n.ID, doc, n.FanIn, n.FanOut)
		}
	}

	b.WriteString("\n## Third-party dependencies\n\nBy import path prefix, with the number of repository packages that import them.\n\n| Module | Packages |\n|---|---:|\n")
	for _, e := range g.External[:min(25, len(g.External))] {
		fmt.Fprintf(&b, "| `%s` | %d |\n", e.Path, e.Packages)
	}
	return b.String()
}

func moduleOf(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1]
	}
	return id
}
