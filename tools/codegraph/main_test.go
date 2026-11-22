package main

import (
	"strings"
	"testing"
)

func fakePackages() []listedPackage {
	m := "example.com/app"
	return []listedPackage{
		{ImportPath: m + "/cmd/app", Doc: "Command app runs it. More text.", GoFiles: []string{"main.go"}, Imports: []string{m + "/internal/db", m + "/modules/news", "fmt", "github.com/x/y/z"}},
		{ImportPath: m + "/internal/db", Doc: "Package db talks to the database.", GoFiles: []string{"db.go"}, Imports: []string{"github.com/x/y/z/sub"}},
		{ImportPath: m + "/modules/news", GoFiles: []string{"news.go"}, Imports: []string{m + "/internal/db"}},
		{ImportPath: m + "/modules/payments", GoFiles: []string{"p.go"}, Imports: []string{m + "/internal/db"}},
		{ImportPath: m + "/internal/empty"},
	}
}

func TestBuildGraph(t *testing.T) {
	g := build("example.com/app", fakePackages())
	if len(g.Nodes) != 3 {
		t.Fatalf("want 3 nodes (add-on and empty packages skipped), got %d", len(g.Nodes))
	}
	byID := map[string]Node{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	db := byID["internal/db"]
	if db.FanIn != 2 || db.FanOut != 0 || db.Layer != "internal" {
		t.Fatalf("db node: %+v", db)
	}
	if got := byID["cmd/app"]; got.FanOut != 2 || got.Doc != "Command app runs it." {
		t.Fatalf("cmd node: %+v", got)
	}
	if len(g.External) != 1 || g.External[0].Path != "github.com/x/y" || g.External[0].Packages != 2 {
		t.Fatalf("external: %+v", g.External)
	}
}

func TestRenderIsDeterministicAndHasSections(t *testing.T) {
	g := build("example.com/app", fakePackages())
	a, err := render(g)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := render(build("example.com/app", fakePackages()))
	for name := range a {
		if string(a[name]) != string(b[name]) {
			t.Fatalf("%s differs between runs", name)
		}
	}
	md := string(a["CODEMAP.md"])
	for _, want := range []string{"## Layers", "flowchart LR", "## Most imported packages", "## Package index", "`internal/db`"} {
		if !strings.Contains(md, want) {
			t.Errorf("CODEMAP.md misses %q", want)
		}
	}
	if strings.Contains(string(a["codegraph.html"]), "</script><script") && strings.Count(string(a["codegraph.html"]), `id="data"`) != 1 {
		t.Error("data block must appear once")
	}
}

func TestFamilyAndLayer(t *testing.T) {
	if family("golang.org/x/sync/singleflight") != "golang.org/x/sync" || family("gopkg.in/yaml.v3") != "gopkg.in/yaml.v3" {
		t.Error("family trimming")
	}
	if layerOf("modules/pages") != "modules" || layerOf("weird/path") != "other" || layerOf(".") != "root" {
		t.Error("layer detection")
	}
}
