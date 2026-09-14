// Command assetguard rejects mixed cache-buster versions per theme asset type.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var assetPattern = regexp.MustCompile(`theme\.(css|js)\?v=([A-Za-z0-9_-]+)`)

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	unused, err := inspectVendorReferences(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(unused) != 0 {
		for _, path := range unused {
			fmt.Fprintf(os.Stderr, "assetguard: embedded vendor asset is not referenced: %s\n", path)
		}
		os.Exit(1)
	}
	versions := map[string]map[string][]string{}
	err = filepath.WalkDir(*root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || (filepath.Ext(path) != ".go" && filepath.Ext(path) != ".html") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range assetPattern.FindAllStringSubmatch(string(data), -1) {
			if versions[match[1]] == nil {
				versions[match[1]] = map[string][]string{}
			}
			versions[match[1]][match[2]] = append(versions[match[1]][match[2]], path)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	types := make([]string, 0, len(versions))
	for typ := range versions {
		types = append(types, typ)
	}
	sort.Strings(types)
	if len(types) == 0 {
		fmt.Fprintln(os.Stderr, "assetguard: no theme assets found")
		os.Exit(1)
	}
	for _, typ := range types {
		keys := make([]string, 0, len(versions[typ]))
		for key := range versions[typ] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		if len(keys) > 1 {
			fmt.Fprintf(os.Stderr, "assetguard: mixed theme.%s versions:\n", typ)
			for _, key := range keys {
				fmt.Fprintf(os.Stderr, " - %s: %s\n", key, strings.Join(versions[typ][key], ", "))
			}
			os.Exit(1)
		}
		fmt.Printf("assetguard: OK theme.%s=%s references=%d\n", typ, keys[0], len(versions[typ][keys[0]]))
	}
}

// inspectVendorReferences ensures every browser runtime asset embedded in the
// binary is reachable from a rendered/runtime source surface. License files
// and placeholders are intentionally excluded from this check.
func inspectVendorReferences(root string) ([]string, error) {
	vendorRoot := filepath.Join(root, "static", "vendor")
	var assets []string
	err := filepath.WalkDir(vendorRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || (filepath.Ext(path) != ".js" && filepath.Ext(path) != ".css") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		assets = append(assets, filepath.ToSlash(rel))
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var sources []string
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" || entry.Name() == ".tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".html" && ext != ".js" && ext != ".css" {
			return nil
		}
		sources = append(sources, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	contents := make([]string, 0, len(sources))
	for _, path := range sources {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		contents = append(contents, string(data))
	}
	var unused []string
	for _, asset := range assets {
		needle := "/" + asset
		found := false
		for _, content := range contents {
			if strings.Contains(content, needle) {
				found = true
				break
			}
		}
		if !found {
			unused = append(unused, asset)
		}
	}
	sort.Strings(unused)
	return unused, nil
}
