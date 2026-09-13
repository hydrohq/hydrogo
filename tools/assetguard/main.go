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
	versions := map[string]map[string][]string{}
	err := filepath.WalkDir(*root, func(path string, entry fs.DirEntry, walkErr error) error {
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
